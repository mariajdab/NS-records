package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
)

func readUrlSuffix(url string) ([]string, int, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, err
	}
	bodyS := string(body)
	r := regexp.MustCompile(`(?m)^(/)(.*)[^/]|^(\n)(/)(.*)[^/]`)
	domains := r.ReplaceAllString(bodyS, "")
	domainsList := strings.Split(domains, "\n")

	return domainsList, len(domainsList), nil
}

func writeReport(name string, domains []string, wg *sync.WaitGroup, file *os.File) {
	defer wg.Done()

	for _, suffix := range domains {
		_, err := net.LookupNS(name + suffix)

		if err == nil {
			_, err = file.WriteString(name + suffix + ": Yes NS \n")
			if err != nil {
				file.WriteString("Could not write to file \n")
			}

		} else {
			e := err.(*net.DNSError)

			if e.IsTimeout {
				_, err = file.WriteString(name + suffix + ": TIMEOUT, please check your connection \n")
				if err != nil {
					file.WriteString("Could not write to file \n")
				}
				continue
			}
			if e.IsNotFound {
				_, err = file.WriteString(name + suffix + ": No NS \n")
				if err != nil {
					file.WriteString("Could not write to file \n")
				}
				continue
			}
			if e.IsTemporary {
				_, err = file.WriteString(name + suffix + ": Temporary \n")
				if err != nil {
					file.WriteString("Could not write to file \n")
				}
				continue
			}
			_, err = file.WriteString(name + suffix + ": Unexpected error \n")
			if err != nil {
				file.WriteString("Could not write to file \n")
			}
		}
	}
}

func launchRoutines(name string, domains []string, file *os.File, lenTotalDomains int) {
	parts := 1000
	partSize := lenTotalDomains / parts
	var wg sync.WaitGroup

	for i := 0; i < parts; i++ {
		wg.Add(1)
		go writeReport(name+".", domains[i*partSize:(i+1)*partSize], &wg, file)
	}
	go writeReport(name+".", domains[parts*partSize:], &wg, file)
	wg.Wait()
}

func main() {

	urlPrefixList := "https://publicsuffix.org/list/public_suffix_list.dat"

	domains, lenTotalDomains, err := readUrlSuffix(urlPrefixList)
	if err != nil {
		log.Println("Could not connect with Public suffix list")
	}

	file, err := os.Create("records1.txt")
	if err != nil {
		log.Fatalf("Failed creating file: %s", err)
	}
	defer file.Close()

	// Pass a name to the function to find all domains with NS records registered
	name := "crypto"
	launchRoutines(name, domains, file, lenTotalDomains)

	fmt.Println("It's done")

}
