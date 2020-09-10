package main

import (
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

/*
 /api?ip=xxxxx (optional, use request ip if not submitted)
 host=foo
 domain=example.com
 key=apikey
*/

/*
 * Record type
 */
type Record struct {
	hostname   string
	domainname string
	accessKey  string
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

func index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Test!")
	log.Printf(GetIP(r))
	for k, v := range r.URL.Query() {
		fmt.Printf("%s: %s\n", k, v)
	}
}

func foo(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "foo")
	fmt.Fprintf(w, "Hi there, I love %s!", r.URL.Path[1:])
}

func handleAPI(w http.ResponseWriter, r *http.Request) {
	keys, ok := r.URL.Query()["key"]

	if !ok || len(keys[0]) < 1 {
		log.Println("Url Param 'key' is missing")
		fmt.Fprintf(w, "error")
		return
	}

	q := r.URL.Query()
	fmt.Println(q["key"])
	fmt.Println(q.Get("host"))
	fmt.Println(q.Get("domain"))

	// Query()["key"] will return an array of items,
	// we only want the single item.
	key := keys[0]

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
		panic(err.Error())
	}
	return db
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

	db := dbConn()

	results, err := db.Query("SELECT r.hostname, d.domainname, d.access_key FROM dynrecords r, domains d WHERE d.id=r.domain_id")
	if err != nil {
		panic(err.Error()) // proper error handling instead of panic in your app
	}

	for results.Next() {
		var record Record
		// for each row, scan the result into our tag composite object
		err = results.Scan(&record.hostname, &record.domainname, &record.accessKey)
		if err != nil {
			panic(err.Error()) // proper error handling instead of panic in your app
		}
		// and then print out the tag's Name attribute

		log.Printf(record.hostname)
		log.Printf(record.domainname)
		log.Printf(record.accessKey)
		//log.Printf(user.ID)
	}

	defer db.Close()

	http.HandleFunc("/", index)

	http.HandleFunc("/foo", foo)
	http.HandleFunc("/api", handleAPI)

	fs := http.FileServer(http.Dir("static/"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.ListenAndServe(":8080", nil)
}
