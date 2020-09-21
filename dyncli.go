package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/geschke/dynpower/cmd"
)

func handleDomainCommand(fs *flag.FlagSet, dsn string) {
	fmt.Println("handle Domain command")
	fmt.Println(fs.Args())
	switch fs.Arg(0) {
	case "list":
		fmt.Println("Command: list")
	case "add":
		fmt.Println("Command: add")
	default:
		fmt.Println("Unknown command.")
	}

}

func handleHostCommand(fs *flag.FlagSet, dsn string) {
	fmt.Println("handle Host command")
	fmt.Println(fs.Args())
	switch fs.Arg(0) {
	case "list":
		fmt.Println("Command: list")
	case "add":
		fmt.Println("Command: add")
	default:
		fmt.Println("Unknown command.")
	}

}

func main() {
	fmt.Println("dyncli is a small helper tool to manage the dynpower database.")

	//flag.StringVar(&dsn, "dsn", "", "MySQL/MariaDB Data Source Name as described in https://github.com/go-sql-driver/mysql#dsn-data-source-name")

	hostCmd := flag.NewFlagSet("host", flag.ExitOnError)
	domainCmd := flag.NewFlagSet("domain", flag.ExitOnError)

	hostDsn := hostCmd.String("dsn", "", "MySQL/MariaDB Data Source Name as described in https://github.com/go-sql-driver/mysql#dsn-data-source-name")
	domainDsn := domainCmd.String("dsn", "", "MySQL/MariaDB Data Source Name as described in https://github.com/go-sql-driver/mysql#dsn-data-source-name")

	/*dbPasswordPtr := flag.String("password", "", "Database password")
	dbHostPtr := flag.String("host", "", "Database server")
	dbNamePtr := flag.String("dbname")
	*/
	//numbPtr := flag.Int("numb", 42, "an int")
	//boolPtr := flag.Bool("fork", false, "a bool")

	//var svar string
	//flag.StringVar(&svar, "svar", "bar", "a string var")

	// todo maybe: use flag subcommands
	flag.Parse()

	switch flag.Arg(0) {
	case "encrypt":
		password := flag.Arg(1)
		if len(password) < 1 {
			fmt.Println("\nPassword parameter missing. \n")
			os.Exit(1)
			//panic(err.Error()) // proper error handling instead of panic in your app
		}
		cmd.Encrypt(password)
	case "domain":
		command := flag.Arg(1)
		if len(command) < 1 {
			fmt.Println("\nManaga domains.\n")
			fmt.Println("Available commands:")
			fmt.Println("\tlist\t\t List domains in database.")
			fmt.Println("\tadd <domain> <access key>\t Add domain with access key to database.\n")

			os.Exit(0)
		}
		domainCmd.Parse(os.Args[2:])
		fmt.Println("subcommand 'domain'")
		fmt.Println("  dsn:", *domainDsn)
		fmt.Println("  tail:", domainCmd.Args())

		handleDomainCommand(domainCmd, *domainDsn)

		fmt.Println("")
	case "host":
		command := flag.Arg(1)
		if len(command) < 1 {
			fmt.Println("\nManaga hosts.\n")
			fmt.Println("Available commands:")
			fmt.Println("\tlist <domain>\t List hosts of <domain> in database.")
			fmt.Println("\tadd <domain> <host>\t Add host of <domain> to database.")

			os.Exit(0)
		}
		//handleHostCommand(os.Args[2:], dsn)
		hostCmd.Parse(os.Args[2:])
		fmt.Println("subcommand 'host'")
		fmt.Println("  dsn:", *hostDsn)
		fmt.Println("  tail:", hostCmd.Args())
		handleHostCommand(hostCmd, *hostDsn)

		fmt.Println("")
	default:
		fmt.Println("Unknown or undefined command, please use the following commands:\n")
		fmt.Println("\tencrypt <password> :\t Encrypt password string to enter into database table.\n")

		fmt.Println("\tdomain -dsn <dsn> <command> <options>:\t Manage domain entries.")
		fmt.Println("\thost -dsn <dsn> <command> <options>:\t Manage host entries.")

		fmt.Println("\n\n")
		os.Exit(0)
		return
	}

}
