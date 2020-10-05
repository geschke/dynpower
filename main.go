package main

import (
	"database/sql"
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

/*
 * DNSRecord type
 */
type DNSRecord struct {
	host      string
	domain    string
	accessKey string
	ip        string
}

// GetIP gets a requests IP address by reading off the forwarded-for
// header (for proxies) and falls back to use the remote address.
// check x-forwarded for later...
func GetIP(r *http.Request) string {
	forwarded := r.Header.Get("X-FORWARDED-FOR")
	if forwarded != "" {
		parsedIP := net.ParseIP(forwarded)
		if parsedIP != nil {
			return net.IP.String(parsedIP)
		}
		ip, _, err := net.SplitHostPort(forwarded)
		if err != nil {
			log.Printf("forwarded for: %s is not IP:port\n", forwarded)
			return "error"
		}
		return ip
	}

	forwarded = r.Header.Get("X-Real-Ip")
	if forwarded != "" {
		parsedIP := net.ParseIP(forwarded)
		if parsedIP != nil {
			return net.IP.String(parsedIP)
		}

		ip, _, err := net.SplitHostPort(forwarded)
		if err != nil {
			log.Printf("X-Real-Ip: %s is not IP:port\n", forwarded)
			return "error"
		}
		return ip
	}

	parsedIP := net.ParseIP(r.RemoteAddr)
	if parsedIP != nil {
		return net.IP.String(parsedIP)
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		log.Printf("userip: %q is not IP:port\n", r.RemoteAddr)
		return "error"
	}
	return ip
}

// go get -u github.com/go-sql-driver/mysql

func handleEverything(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Unknown request")
	log.Printf(GetIP(r))
	//for k, v := range r.URL.Query() {
	//		fmt.Printf("%s: %s\n", k, v)
	//}
}

func foo(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "foo")
	fmt.Fprintf(w, "Hi there, I love %s!", r.URL.Path[1:])
}

func validateRequest(dnsrecord DNSRecord) bool {
	//log.Println(dnsrecord)
	db := dbConn()
	var count int

	// 1. check api key and domain

	var hashedPassword string
	err := db.QueryRow("SELECT d.access_key FROM domains d WHERE d.domainname=?", dnsrecord.domain).Scan(&hashedPassword)

	if err != nil {
		log.Println("Database query problem: " + err.Error())
		return false
	}

	//fmt.Printf("access_key from database: %s\n", hashedPassword)

	accessKeyBytes := []byte(dnsrecord.accessKey) // convert submitted api key into []byte
	hashedPasswordBytes := []byte(hashedPassword) // the hashed password from database has to be converted, too

	err = bcrypt.CompareHashAndPassword(hashedPasswordBytes, accessKeyBytes)
	if err != nil {
		log.Println("Wrong API key")
		return false
	}

	log.Println("Now get host and domain")
	//fmt.Println(err) // nil means it is a match

	// 2. check if host entry exists

	err = db.QueryRow("SELECT count(*) as cnt FROM dynrecords r, domains d WHERE d.id=r.domain_id AND r.hostname=? AND d.domainname=? AND d.access_key=?", dnsrecord.host, dnsrecord.domain, hashedPassword).Scan(&count)

	switch {
	case err != nil:
		log.Println(err)
	default:
		fmt.Printf("Number of rows are %d\n", count)
	}

	defer db.Close()
	if count != 1 {
		return false
	}
	return true

}

func updateSoa(dnsrecord DNSRecord) {
	// get SOA entry
	log.Printf("get SOA entry from records")
	db := dbConnPdns()

	var content string
	err := db.QueryRow("SELECT content FROM records WHERE type=? AND name=?", "SOA", dnsrecord.domain).Scan(&content)
	if err != nil {
		log.Println("Database problem: " + err.Error())
		//os.Exit(1)
		//panic(err.Error()) // proper error handling instead of panic in your app
	}

	// split content into its parts, increase serial
	// to test: is it necessary to update serial in domains table?
	// check: will the NOTIFY request perform? Local tests seem strange now...

	var contentParts = strings.Split(content, " ")

	log.Println(contentParts)
	log.Println(contentParts[2])
	var serial, _ = strconv.ParseInt(contentParts[2], 10, 64)
	serial = serial + 2
	log.Println(serial)
	var serialString = strconv.FormatInt(int64(serial), 10)
	log.Println(serialString)

	contentParts[2] = serialString
	log.Println(contentParts)

	var contentModified = strings.Join(contentParts, " ")

	log.Println(contentModified)

	updateStmt, err := db.Prepare("UPDATE records SET content=? WHERE type=? AND name=?")
	if err != nil {
		log.Println("Update problem: " + err.Error())
		//os.Exit(1)

	}
	_, err = updateStmt.Exec(contentModified, "SOA", dnsrecord.domain)
	if err != nil {
		log.Println("Update problem: " + err.Error())
	}

	defer db.Close()

}

func updateRecord(dnsrecord DNSRecord) {
	// get SOA entry

	db := dbConnPdns()

	updateStmt, err := db.Prepare("UPDATE records SET content=? WHERE type=? AND name=?")
	if err != nil {
		log.Println("Update problem: " + err.Error())
		//os.Exit(1)

	}
	_, err = updateStmt.Exec(dnsrecord.ip, "A", dnsrecord.host+"."+dnsrecord.domain)
	if err != nil {
		log.Println("Update problem: " + err.Error())
	}

	defer db.Close()

}

func updateDynRecords(dnsrecord DNSRecord) {
	db := dbConn()

	updateStmt, err := db.Prepare("UPDATE dynrecords r, domains d SET r.host_updated=now() WHERE r.hostname=? AND r.domain_id=d.id AND d.domainname=?")
	if err != nil {
		log.Println("Update problem: " + err.Error())
		//os.Exit(1)

	}
	_, err = updateStmt.Exec(dnsrecord.host, dnsrecord.domain)

	if err != nil {
		log.Println("Update problem: " + err.Error())
	}

	defer db.Close()

}

func updateEntry(dnsrecord DNSRecord) {
	//log.Println(dnsrecord)

	if !validateRequest(dnsrecord) {
		log.Println("Invalid request data, please check host, domain and key!")
		return
	}
	log.Println("Data valid, now update!")

	updateRecord(dnsrecord)
	updateSoa(dnsrecord)
	updateDynRecords(dnsrecord)
	log.Println("Records updated")

}

func handleUpdate(w http.ResponseWriter, r *http.Request) {
	//keys, ok := r.URL.Query()["key"]

	//log.Println(ok)

	/*if !ok || len(keys[0]) < 1 {
		log.Println("Url Param 'key' is missing")
		fmt.Fprintf(w, "error")
		return
	}*/

	q := r.URL.Query()

	//fmt.Println(q["key"])

	key := q.Get("key")
	if len(key) < 1 {
		log.Println("API Key is missing")
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

	// todo get ip from query string or from request

	//fmt.Println(domain)

	var record DNSRecord
	record.host = host
	record.domain = domain
	record.accessKey = key

	ip := q.Get("ip") // prefer submitted IP over request IP
	if len(ip) < 1 {
		ip = GetIP(r)
		if ip == "error" {
			log.Println("Could not get IP address, exiting...")
			return
		}

	} else if net.ParseIP(ip) == nil {
		log.Println("Invalid IP, exiting...")
		return
	}
	record.ip = ip

	log.Println("ok, we'll try an update!")
	updateEntry(record)

	// Query()["key"] will return an array of items,
	// we only want the single item.
	//key := keys[0]

	//log.Println("Url Param 'key' is: " + string(key))
	fmt.Fprintf(w, "ok")
}

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

/*
 *	Check database connection by performing a query, exit in error case
 */
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

/*
 *	Check PowerDNS  database connection by performing a query, exit in error case
 */
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

	log.Println("Server started...")

	http.HandleFunc("/", handleEverything)
	http.HandleFunc("/api", handleEverything)

	//http.HandleFunc("/foo", foo)
	http.HandleFunc("/api/update", handleUpdate)

	//fs := http.FileServer(http.Dir("static/"))
	//http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.ListenAndServe(":8080", nil)
}
