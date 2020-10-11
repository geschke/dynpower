package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
)

/*
 /api?ip=xxxxx (optional, use request ip if not submitted)
 host=foo
 domain=example.com
 key=apikey

*/

// DNSRecord is a struct to collect the data collected by request to update the database
type DNSRecord struct {
	host      string
	domain    string
	accessKey string
	ip        string
}

// GetIP gets a requests IP address by reading off the forwarded-for
// header (for proxies) and falls back to use the remote address.
// check x-forwarded for later...
func GetIP(r *http.Request) (string, error) {
	forwarded := r.Header.Get("X-FORWARDED-FOR")
	if forwarded != "" {
		parsedIP := net.ParseIP(forwarded)
		if parsedIP != nil {
			return net.IP.String(parsedIP), nil
		}
		ip, _, err := net.SplitHostPort(forwarded)
		if err != nil {
			log.Printf("forwarded for: %s is not IP:port\n", forwarded)
			return "", errors.New(err.Error())
		}
		return ip, nil
	}

	forwarded = r.Header.Get("X-Real-Ip")
	if forwarded != "" {
		parsedIP := net.ParseIP(forwarded)
		if parsedIP != nil {
			return net.IP.String(parsedIP), nil
		}

		ip, _, err := net.SplitHostPort(forwarded)
		if err != nil {
			log.Printf("X-Real-Ip: %s is not IP:port\n", forwarded)
			return "", errors.New(err.Error())
		}
		return ip, nil
	}

	parsedIP := net.ParseIP(r.RemoteAddr)
	if parsedIP != nil {
		return net.IP.String(parsedIP), nil
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		log.Printf("userip: %q is not IP:port\n", r.RemoteAddr)
		return "", errors.New(err.Error())

	}
	return ip, nil
}

// Handle all requests which do not match
func handleEverything(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Unknown request")
	ip, err := GetIP(r)
	if err != nil {
		log.Printf("Unknown request from unknown IP address")
		return
	}
	log.Printf("Unknown request from IP " + ip)
	return
}

// Validate domain and access key
func validateRequest(dnsrecord DNSRecord) (bool, error) {
	//log.Println(dnsrecord)
	db := dbConn()
	var count int

	// 1. check api key and domain

	var hashedPassword string
	err := db.QueryRow("SELECT d.access_key FROM domains d WHERE d.domainname=?", dnsrecord.domain).Scan(&hashedPassword)

	defer db.Close()
	if err != nil {
		log.Println("Database query problem: " + err.Error())
		return false, errors.New("Database query problem: " + err.Error())

	}

	accessKeyBytes := []byte(dnsrecord.accessKey) // convert submitted api key into []byte
	hashedPasswordBytes := []byte(hashedPassword) // the hashed password from database has to be converted, too

	err = bcrypt.CompareHashAndPassword(hashedPasswordBytes, accessKeyBytes)
	if err != nil {
		log.Println("Wrong API key")
		return false, nil // no error, but wrong API key
	}

	// 2. check if host entry exists

	err = db.QueryRow("SELECT count(*) as cnt FROM dynrecords r, domains d WHERE d.id=r.domain_id AND r.hostname=? AND d.domainname=? AND d.access_key=?", dnsrecord.host, dnsrecord.domain, hashedPassword).Scan(&count)

	if err != nil {
		log.Println("Database query problem: " + err.Error())
		return false, errors.New("Database query problem: " + err.Error())
	}

	if count != 1 {
		return false, errors.New("Wrong number of hosts found")
	}
	return true, nil

}

// Update SOA entry in PowerDNS records table
func updateSoa(dnsrecord DNSRecord) (bool, error) {
	// get SOA entry
	log.Printf("get SOA entry from records")
	db := dbConnPdns()

	var content string
	err := db.QueryRow("SELECT content FROM records WHERE type=? AND name=?", "SOA", dnsrecord.domain).Scan(&content)
	if err != nil {
		log.Println("Database problem: " + err.Error())
		return false, errors.New(err.Error())
	}
	defer db.Close()

	// split content into its parts, increase serial to perform AXFR

	var contentParts = strings.Split(content, " ")

	//log.Println(contentParts)
	//log.Println(contentParts[2])
	var serial, _ = strconv.ParseInt(contentParts[2], 10, 64)
	serial = serial + 2
	//log.Println(serial)
	var serialString = strconv.FormatInt(int64(serial), 10)
	//log.Println(serialString)

	contentParts[2] = serialString
	//log.Println(contentParts)

	var contentModified = strings.Join(contentParts, " ")

	//log.Println(contentModified)

	updateStmt, err := db.Prepare("UPDATE records SET content=? WHERE type=? AND name=?")
	if err != nil {
		log.Println("Update problem: " + err.Error())
		return false, errors.New(err.Error())
		//os.Exit(1)

	}
	_, err = updateStmt.Exec(contentModified, "SOA", dnsrecord.domain)
	if err != nil {
		log.Println("Update problem: " + err.Error())
		return false, errors.New(err.Error())
	}
	return true, nil

}

// Update host in PowerDNS record table with new IP address
func updateRecord(dnsrecord DNSRecord) (bool, error) {
	// get SOA entry

	db := dbConnPdns()

	defer db.Close()

	updateStmt, err := db.Prepare("UPDATE records SET content=? WHERE type=? AND name=?")
	if err != nil {
		log.Println("Prepare record update problem: " + err.Error())
		return false, errors.New("Prepare record update problem: " + err.Error())

	}
	_, err = updateStmt.Exec(dnsrecord.ip, "A", dnsrecord.host+"."+dnsrecord.domain)
	if err != nil {
		log.Println("Update problem: " + err.Error())
		return false, errors.New("Update record problem: " + err.Error())

	}

	return true, nil
}

// Update dynrecords table with new timestamp of last update
func updateDynRecords(dnsrecord DNSRecord) (bool, error) {
	db := dbConn()

	defer db.Close()

	updateStmt, err := db.Prepare("UPDATE dynrecords r, domains d SET r.host_updated=now() WHERE r.hostname=? AND r.domain_id=d.id AND d.domainname=?")
	if err != nil {
		log.Println("Prepare update problem: " + err.Error())
		return false, errors.New("Prepare update problem: " + err.Error())
		//os.Exit(1)

	}
	_, err = updateStmt.Exec(dnsrecord.host, dnsrecord.domain)

	if err != nil {
		log.Println("Update problem: " + err.Error())
		return false, errors.New("Update problem: " + err.Error())
	}
	return true, nil
}

// Update PowerDNS and dynpower database after validating request
func updateEntry(dnsrecord DNSRecord) (bool, error) {
	//log.Println(dnsrecord)

	var err error

	requestValid, err := validateRequest(dnsrecord)

	if err != nil {
		log.Println("Error by validating request data " + err.Error())
		return false, errors.New("Error by validating request data " + err.Error())
	}

	if requestValid == false {
		log.Println("Invalid request data, please check host, domain and key")
		return false, errors.New("Invalid request data, please check host, domain and key")

	}
	log.Println("Data valid, now update!")

	_, err = updateRecord(dnsrecord)
	if err != nil {
		log.Println("Error by updating records entry")
		return false, errors.New("Error by updating records entry")
	}
	_, err = updateSoa(dnsrecord)
	if err != nil {
		log.Println("Error by updating SOA entry")
		return false, errors.New("Error by updating SOA entry")
	}
	_, err = updateDynRecords(dnsrecord)
	if err != nil {
		log.Println("Error by updating dynrecords entry")
		return false, errors.New("Error by updating dynrecords entry")
	}

	log.Println("All records updated")
	return true, nil
}

// Handle the update request: get request data and call update function in case of no error
func handleUpdate(w http.ResponseWriter, r *http.Request) {

	var err error
	var record DNSRecord

	q := r.URL.Query()

	//fmt.Println(q["key"])

	key := q.Get("key")
	if len(key) < 1 {
		log.Println("Missing API key")
		fmt.Fprintf(w, "error")
		return
	}
	//log.Println(key)

	host := q.Get("host")
	if len(host) < 1 {
		log.Println("Host entry is missing")
		fmt.Fprintf(w, "error")
		return
	}
	//fmt.Println(host)

	domain := q.Get("domain")
	if len(domain) < 1 {
		log.Println("Domain entry is missing")
		fmt.Fprintf(w, "error")
		return
	}

	// get ip from query string or from request

	record.host = host
	record.domain = domain
	record.accessKey = key

	// todo maybe: use HTTP error codes

	ip := q.Get("ip") // prefer submitted IP over request IP
	if len(ip) < 1 {
		ip, err = GetIP(r)
		if err != nil {
			log.Println("Could not get IP address, exiting...")
			fmt.Fprintf(w, "An error occurred, see log entry.")
			return
		}

	} else if net.ParseIP(ip) == nil {
		log.Println("Invalid IP, exiting...")
		fmt.Fprintf(w, "An error occurred, see log entry.")
		return
	}
	record.ip = ip

	log.Println("Try to update record...")
	_, err = updateEntry(record)
	if err != nil {
		log.Println("Error: " + err.Error())
		fmt.Fprintf(w, "An error occurred, see log entry.")
		return
	}

	fmt.Fprintf(w, "ok")
	return

}

// Create connection to dynpower database
func dbConn() (db *sql.DB) {
	dbname := os.Getenv("DBNAME")
	dbhost := os.Getenv("DBHOST")
	dbuser := os.Getenv("DBUSER")
	dbpassword := os.Getenv("DBPASSWORD")

	db, err := sql.Open("mysql", dbuser+":"+dbpassword+"@tcp("+dbhost+":3306)/"+dbname)
	if err != nil {
		log.Println("Error by connecting database.")
		panic(err.Error())
	}
	return db
}

// Create connection to PowerDNS database
func dbConnPdns() (db *sql.DB) {
	dbname := os.Getenv("PDNS_DBNAME")
	dbhost := os.Getenv("PDNS_DBHOST")
	dbuser := os.Getenv("PDNS_DBUSER")
	dbpassword := os.Getenv("PDNS_DBPASSWORD")

	db, err := sql.Open("mysql", dbuser+":"+dbpassword+"@tcp("+dbhost+":3306)/"+dbname)
	if err != nil {
		log.Println("Error by connecting database.")
		panic(err.Error())
	}
	return db
}

// Check database connection by performing a query, exit in error case
func checkDb() {
	db := dbConn()
	_, err := db.Query("SELECT r.hostname, d.domainname, d.access_key FROM dynrecords r, domains d WHERE d.id=r.domain_id")
	if err != nil {
		log.Println("Database problem: " + err.Error())
		os.Exit(1)
		//panic(err.Error()) // proper error handling instead of panic in your app
	}
	return
}

// Check PowerDNS database connection by performing a query, exit in error case
func checkDbPdns() {
	db := dbConnPdns()
	_, err := db.Query("SELECT * FROM domains")
	if err != nil {
		log.Println("PowerDNS database problem: " + err.Error())
		os.Exit(1)
		//panic(err.Error()) // proper error handling instead of panic in your app
	}
	return
}

func main() {

	checkDb()
	checkDbPdns()

	log.Println("dynpower server started...")

	http.HandleFunc("/", handleEverything)
	http.HandleFunc("/api", handleEverything)

	http.HandleFunc("/api/update", handleUpdate)

	http.ListenAndServe(":8080", nil)
}
