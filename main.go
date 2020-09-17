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
}

// GetIP gets a requests IP address by reading off the forwarded-for
// header (for proxies) and falls back to use the remote address.
// check x-forwarded for later...
func GetIP(r *http.Request) string {
	forwarded := r.Header.Get("X-FORWARDED-FOR")
	if forwarded != "" {
		ip, _, err := net.SplitHostPort(forwarded)
		if err != nil {
			fmt.Printf("forwarded for: %q is not IP:port", r.RemoteAddr)
			return "error"
		}

		return ip
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		fmt.Printf("userip: %q is not IP:port", r.RemoteAddr)
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
	log.Println(dnsrecord)
	db := dbConn()
	var count int

	err := db.QueryRow("SELECT count(*) as cnt FROM dynrecords r, domains d WHERE d.id=r.domain_id AND r.hostname=? AND d.domainname=? AND d.access_key=?", dnsrecord.host, dnsrecord.domain, dnsrecord.accessKey).Scan(&count)

	switch {
	case err != nil:
		log.Fatal(err)
	default:
		fmt.Printf("Number of rows are %d\n", count)
	}

	defer db.Close()
	if count != 1 {
		return false
	} else {
		return true
	}
}

func updateSoa(dnsrecord DNSRecord) {
	// get SOA entry
	log.Printf("get SOA entry from records")
	db := dbConnPdns()

	results, err := db.Query("SELECT content FROM records WHERE type=? AND name=?", "SOA", dnsrecord.domain)

	if err != nil {
		log.Println("Database problem." + err.Error())
		os.Exit(1)
		//panic(err.Error()) // proper error handling instead of panic in your app
	}

	for results.Next() {
		var content string
		// for each row, scan the result into our tag composite object
		err = results.Scan(&content)
		if err != nil {
			panic(err.Error()) // proper error handling instead of panic in your app
		}
		// and then print out the tag's Name attribute

		log.Printf(content)
		// split content into its parts, increase serial
		// to test: is it necessary to update serial in domains table?
		// check: will the NOTIFY request perform? Local tests seem strange now...

		var f = strings.Split(content, " ")

		log.Println(f)
		log.Println(f[2])
		var serial, _ = strconv.ParseInt(f[2], 10, 64)
		serial++
		log.Println(serial)
		var serialString = strconv.FormatInt(int64(serial), 10)
		log.Println(serialString)

		f[2] = serialString
		log.Println(f)

		var contentModified = strings.Join(f, " ")

		log.Println(contentModified)
	}

	defer db.Close()

}

func updateEntry(dnsrecord DNSRecord) {
	log.Println(dnsrecord)

	/*
		if err != nil {
			panic(err.Error())
		}
		for query.Next() {
			var r DNSRecord
			// for each row, scan the result into our tag composite object
			err = query.Scan(&r.host, &r.domain, &r.accessKey)
			if err != nil {
				panic(err.Error()) // proper error handling instead of panic in your app
			}
			// and then print out the tag's Name attribute

			log.Printf(r.host)
			log.Printf(r.domain)
			log.Printf(r.accessKey)
			//log.Printf(user.ID)
		}
	*/
	if !validateRequest(dnsrecord) {
		log.Println("Invalid request data, please check host, domain and key!")
		return
	}
	log.Println("Data valid, now update!")

	updateSoa(dnsrecord)

}

func handleAPI(w http.ResponseWriter, r *http.Request) {
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
	log.Println(key)

	host := q.Get("host")
	if len(host) < 1 {
		log.Println("Host entry is missing")
		fmt.Fprintf(w, "error")
		return
	}
	fmt.Println(host)

	domain := q.Get("domain")
	if len(domain) < 1 {
		log.Println("Domain entry is missing")
		fmt.Fprintf(w, "error")
		return
	}

	fmt.Println(domain)

	action := q.Get("action")
	if len(action) < 1 {
		log.Println("Action is missing")
		fmt.Fprintf(w, "error")
		return
	}

	fmt.Println(q.Get("action")) // currently only: update

	switch action {
	case "update":
		var record DNSRecord
		record.host = host
		record.domain = domain
		record.accessKey = key

		log.Println("ok, we'll try an update!")
		updateEntry(record)
	default:
		log.Println("This action is currently undefined")
	}

	// Query()["key"] will return an array of items,
	// we only want the single item.
	//key := keys[0]

	log.Println("Url Param 'key' is: " + string(key))
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

	/*fmt.Println("DBNAME: ", os.Getenv("DBNAME"))
	fmt.Println("DBHOST:", os.Getenv("DBHOST"))
	fmt.Println("DBUSER:", os.Getenv("DBUSER"))
	fmt.Println("DBPASSWORD:", os.Getenv("DBPASSWORD"))

	fmt.Println()
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		fmt.Println(pair[0])
	}*/

	/*	db := dbConn()

		results, err := db.Query("SELECT r.hostname, d.domainname, d.access_key FROM dynrecords r, domains d WHERE d.id=r.domain_id")
		if err != nil {
			log.Println("Database problem." + err.Error())
			os.Exit(1)
			//panic(err.Error()) // proper error handling instead of panic in your app
		}

		for results.Next() {
			var record DNSRecord
			// for each row, scan the result into our tag composite object
			err = results.Scan(&record.host, &record.domain, &record.accessKey)
			if err != nil {
				panic(err.Error()) // proper error handling instead of panic in your app
			}
			// and then print out the tag's Name attribute

			log.Printf(record.host)
			log.Printf(record.domain)
			log.Printf(record.accessKey)
			//log.Printf(user.ID)
		}

		defer db.Close()
	*/

	checkDb()
	checkDbPdns()

	log.Println("Server started...")

	http.HandleFunc("/", handleEverything)

	//http.HandleFunc("/foo", foo)
	http.HandleFunc("/api", handleAPI)

	//fs := http.FileServer(http.Dir("static/"))
	//http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.ListenAndServe(":8080", nil)
}
