package database

import (
    "database/sql"
    "log"
	"fmt"
	"net"
	"time"
	"slices"

    _ "github.com/mattn/go-sqlite3"
)

type Lease struct {
	ID			int
	IP			string
	MAC			string
	LeaseLen	int
	LeasedOn	string
}

func SetupDatabase() {
	db, err := sql.Open("sqlite3", "./leases.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	leasesTableSQL := `CREATE TABLE IF NOT EXISTS leases (
        "id" INTEGER PRIMARY KEY AUTOINCREMENT,
        "mac" TEXT,
        "ip" TEXT,
		"lease_len" INTEGER,
		"leased_on" TEXT,
    );`

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(leasesTableSQL)
    if err != nil {
        log.Fatal(err)
    }

	log.Println("Created/Opened leases.db, created leases table, all success")
}

func CreateLeasesTable(db *sql.DB) (error) {

	leasesTableSQL := `CREATE TABLE IF NOT EXISTS leases (
        "id" INTEGER PRIMARY KEY AUTOINCREMENT,
        "mac" TEXT,
        "ip" TEXT
    );`

	_, err := db.Exec(leasesTableSQL)
    if err != nil {
        return fmt.Errorf("Error creating leases table: %v\n", err)
    }

	return nil
} 

func ConnectDatabase() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "./leases.db")
    if err != nil {
        return nil, fmt.Errorf("Error occured when connecting to leases.db: %v", err)
    }

    err = db.Ping()
    if err != nil {
        return nil, fmt.Errorf("Error occured when pinging db...%v", err)
    }

	err = CreateLeasesTable(db)
	if err != nil {
		return nil, fmt.Errorf("%v\n", err)
	}

	return db, nil
}

func IsExpired(leaseLen int, leasedOn string) (bool) {
	format := "2006-01-02 15:04:05"

	leasedOnTime, err := time.Parse(format, leasedOn)
	if err != nil {
		return true
	}

	timeSince := time.Since(leasedOnTime)

	if int(timeSince.Seconds()) >= leaseLen {
		return true
	}

	return false
}

func IsIPAvailable(db *sql.DB, ip net.IP) (bool) {
	var lease Lease

    query := "SELECT ip, lease_len, leased_on FROM leases WHERE ip = ?"
    err := db.QueryRow(query, ip.String()).Scan(&lease.IP, &lease.LeaseLen, &lease.LeasedOn)
    if err != nil {
        if err == sql.ErrNoRows {
            log.Println("No lease found for that IP")
            return true
        }
        return false
    }

	if IsExpired(lease.LeaseLen, lease.LeasedOn) {
		return true
	}
    return false
}

func IsMACLeased(db *sql.DB, mac net.HardwareAddr) (net.IP) {
	var lease Lease

    query := "SELECT ip, mac, lease_len, leased_on FROM leases WHERE mac = ?"
    err := db.QueryRow(query, mac.String()).Scan(&lease.IP, &lease.MAC, &lease.LeaseLen, &lease.LeasedOn)
    if err != nil {
        if err == sql.ErrNoRows {
            log.Println("MAC does not have a lease")
			return nil        
		}
		// Actual error return
        return nil
    }

	if IsExpired(lease.LeaseLen, lease.LeasedOn) {
		return nil
	}

    return net.ParseIP(lease.IP)
}

func LeaseIP(db *sql.DB, ip net.IP, mac net.HardwareAddr, leaseLen int, leasedOn string) (error) {
	return nil
}

func UnleaseIP(db *sql.DB, ip net.IP) (error) {
	return nil
}

func GenerateIP(db *sql.DB) (net.IP, error) {
    query := "SELECT id, ip FROM leases"

    rows, err := db.Query(query)
    if err != nil {
        return nil, fmt.Errorf("%w\n", err)
    }
    defer rows.Close()

    var ips []net.IP
    for rows.Next() {
        var lease Lease
        err = rows.Scan(&lease.ID, &lease.IP)
        if err != nil {
			if err == sql.ErrNoRows {
				break
			}
            return nil, fmt.Errorf("%w\n", err)
        }

		log.Printf("Leased IP: %v\n", lease.IP)

        ips = append(ips, net.ParseIP(lease.IP))
    }

	startIP := net.IP{192, 168, 1, 20}.To4()
	endIP := net.IP{192, 168, 1, 220}.To4()

	for ip := startIP; !IsIPEqual(ip, endIP); ip = IncrementIP(ip) {
		if !IPsContains(ips, ip) {
			return ip, nil 
		}
	}

	return nil, fmt.Errorf("Unable to generate IP addr, pool full?")
}

func IPsContains(ips []net.IP, ip net.IP) bool {
	for _, item := range ips {
		if slices.Compare(item, ip) == 0 {
			return true
		}
		log.Println(item.String())
	}

	return false
}

// // Function to iterate over the IP pool
// func iterateIPPool(startIP, endIP string) {
// 	// Parse the start and end IPs
// 	start := net.ParseIP(startIP).To4()
// 	end := net.ParseIP(endIP).To4()

// 	if start == nil || end == nil {
// 		fmt.Println("Invalid IP address")
// 		return
// 	}

// 	// Iterate from start IP to end IP
// 	for ip := start; !ipEqual(ip, end); ip = incrementIP(ip) {
// 		fmt.Println(ip)
// 	}

// 	// Print the last IP
// 	fmt.Println(end)
// }

// // Function to increment the last octet of the IP address
func IncrementIP(ip net.IP) net.IP {
	newIP := make(net.IP, len(ip))
	copy(newIP, ip)

	for i := len(newIP) - 1; i >= 0; i-- {
		newIP[i]++
		if newIP[i] != 0 {
			break
		}
	}
	return newIP
}

func IsIPEqual(ip1, ip2 net.IP) bool {
	return ip1.Equal(ip2)
}