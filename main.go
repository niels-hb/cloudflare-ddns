package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/cloudflare/cloudflare-go"
)

func main() {
	cloudflareApiToken := getRequiredEnvironmentVariable("DDNS_CLOUDFLARE_API_TOKEN")
	cloudflareZoneName := getRequiredEnvironmentVariable("DDNS_CLOUDFLARE_ZONE_NAME")
	cloudflareDdnsSubdomain := getRequiredEnvironmentVariable("DDNS_CLOUDFLARE_SUBDOMAIN")
	cloudflareDdnsComment := getRequiredEnvironmentVariable("DDNS_CLOUDFLARE_COMMENT")
	publicIpEndpoint := getOptionalEnvironmentVariable("DDNS_PUBLIC_IP_ENDPOINT", "https://ipinfo.io/ip")

	log.Print("Running Cloudflare DDNS with the following parameters:")
	log.Printf("- API Token: REDACTED")
	log.Printf("- Zone Name: %s", cloudflareZoneName)
	log.Printf("- Subdomain: %s", cloudflareDdnsSubdomain)
	log.Printf("- Comment: %s", cloudflareDdnsComment)
	log.Printf("- Public IP Endpoint: %s", publicIpEndpoint)

	api, err := cloudflare.NewWithAPIToken(cloudflareApiToken)
	if err != nil {
		log.Fatal(err)
	}

	zoneId, err := api.ZoneIDByName(cloudflareZoneName)
	if err != nil {
		log.Fatal(err)
	}

	publicIp := getPublicIp(publicIpEndpoint)
	log.Printf("Current public IP address is: %s", publicIp)

	log.Printf("Searching existing records...")
	allRecords, _, err := api.ListDNSRecords(context.Background(), cloudflare.ZoneIdentifier(zoneId), cloudflare.ListDNSRecordsParams{
		Type: "A",
		Name: fmt.Sprintf("%s.%s", cloudflareDdnsSubdomain, cloudflareZoneName),
	})
	if err != nil {
		log.Fatal(err)
	}

	if len(allRecords) > 1 {
		log.Fatalf("Found multiple records for search A %s. Aborting.", cloudflareDdnsSubdomain)
	} else if len(allRecords) == 1 {
		currentRecord := allRecords[0]
		log.Printf("Found existing record: %s %s -> %s. Will update...", currentRecord.Type, currentRecord.Name, currentRecord.Content)

		record, err := api.UpdateDNSRecord(context.Background(), cloudflare.ZoneIdentifier(zoneId), cloudflare.UpdateDNSRecordParams{
			ID:      currentRecord.ID,
			Type:    "A",
			Name:    cloudflareDdnsSubdomain,
			Content: publicIp,
			Comment: &cloudflareDdnsComment,
		})
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("Sucessfully updated record: %s %s -> %s", record.Type, record.Name, record.Content)
	} else {
		log.Printf("Record doesn't exist yet. Will create...")

		record, err := api.CreateDNSRecord(context.Background(), cloudflare.ZoneIdentifier(zoneId), cloudflare.CreateDNSRecordParams{
			Type:    "A",
			Name:    cloudflareDdnsSubdomain,
			Content: publicIp,
			Comment: cloudflareDdnsComment,
		})
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("Sucessfully created record: %s %s -> %s", record.Type, record.Name, record.Content)
	}
}

func getPublicIp(endpoint string) string {
	req, err := http.Get(endpoint)
	if err != nil {
		return err.Error()
	}
	defer req.Body.Close()

	body, err := io.ReadAll(req.Body)
	if err != nil {
		return err.Error()
	}

	return string(body)
}

func getRequiredEnvironmentVariable(name string) string {
	value := os.Getenv(name)
	if len(value) == 0 {
		log.Fatalf("Expected environment variable %s not found or not set.", name)
	}

	return value
}

func getOptionalEnvironmentVariable(name string, defaultValue string) string {
	value := os.Getenv(name)
	if len(value) == 0 {
		value = defaultValue
	}

	return value
}
