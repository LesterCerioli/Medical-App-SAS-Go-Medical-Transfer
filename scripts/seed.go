package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
)

const (
	dbHost     = "localhost" // Or "db" if running this script from within a Docker container in the same network
	dbPort     = 5432
	dbUser     = "myuser"
	dbPassword = "mypassword"
	dbName     = "mydatabase"
	sslMode    = "disable" // Ensure this matches your PostgreSQL server configuration
)

type Patient struct {
	Name        string
	DateOfBirth string // YYYY-MM-DD
	Email       string
}

type MedicalRecordData struct {
	VisitDate string `json:"visit_date"`
	Doctor    string `json:"doctor"`
	Diagnosis string `json:"diagnosis"`
	Treatment string `json:"treatment"`
	Notes     string `json:"notes,omitempty"`
}

func main() {

	dbHostToUse := dbHost
	if os.Getenv("DOCKER_COMPOSE_NETWORK") == "true" {
		dbHostToUse = "db" // Service name in docker-compose
	}

	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		dbHostToUse, dbPort, dbUser, dbPassword, dbName, sslMode)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Error opening database connection: %v", err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Fatalf("Error pinging database: %v. Ensure the database is running and accessible on %s:%d.", err, dbHostToUse, dbPort)
	}

	fmt.Println("Successfully connected to the database.")

	patients := []Patient{
		{"Alice Wonderland", "1990-05-15", "alice.wonderland@example.com"},
		{"Bob The Builder", "1985-11-20", "bob.builder@example.com"},
		{"Charlie Brown", "2005-02-10", "charlie.brown@example.com"},
		{"Diana Prince", "1970-03-22", "diana.prince@example.com"},
		{"Edward Scissorhands", "1992-07-30", "edward.s@example.net"},
	}

	log.Println("Cleaning existing medical records and patients data...")
	if _, err := db.Exec("DELETE FROM medical_records"); err != nil {
		log.Fatalf("Error cleaning medical_records table: %v", err)
	}
	if _, err := db.Exec("DELETE FROM patients"); err != nil {
		log.Fatalf("Error cleaning patients table: %v", err)
	}
	log.Println("Existing data cleaned.")

	for _, p := range patients {
		var patientID string

		err := db.QueryRow(
			"INSERT INTO patients (name, date_of_birth, email) VALUES ($1, $2, $3) RETURNING id",
			p.Name, p.DateOfBirth, p.Email,
		).Scan(&patientID)

		if err != nil {
			log.Printf("Error inserting patient %s: %v", p.Name, err)
			continue // Skip to next patient if this one fails
		}
		fmt.Printf("Inserted patient: %s (ID: %s)\n", p.Name, patientID)

		medicalRecords := []MedicalRecordData{
			{
				VisitDate: time.Now().AddDate(0, -int(time.Month(patientID[0]%6+1)), -int(patientID[1]%28)).Format("2006-01-02"), // Vary visit date
				Doctor:    "Dr. Smith",
				Diagnosis: "Common Cold",
				Treatment: "Rest and fluids",
				Notes:     "Follow up in a week if symptoms persist.",
			},
			{
				VisitDate: time.Now().AddDate(0, -int(time.Month(patientID[2]%3+1)), -int(patientID[3]%28)).Format("2006-01-02"), // Vary visit date
				Doctor:    "Dr. Jones",
				Diagnosis: "Routine Checkup",
				Treatment: "N/A",
				Notes:     "All clear. Next checkup in 1 year.",
			},
		}
		if p.Name == "Alice Wonderland" { // Add more specific records for one patient
			medicalRecords = append(medicalRecords, MedicalRecordData{
				VisitDate: time.Now().AddDate(0, 0, -5).Format("2006-01-02"),
				Doctor:    "Dr. Hyde",
				Diagnosis: "Sprained Ankle",
				Treatment: "Ice and elevation",
			})
		}

		for i, mrData := range medicalRecords {
			jsonData, err := json.Marshal(mrData)
			if err != nil {
				log.Printf("Error marshalling medical record data for patient %s: %v", p.Name, err)
				continue
			}

			_, err = db.Exec(
				"INSERT INTO medical_records (patient_id, record_data) VALUES ($1, $2)",
				patientID, jsonData,
			)
			if err != nil {
				log.Printf("Error inserting medical record %d for patient %s: %v", i+1, p.Name, err)
				continue
			}
			fmt.Printf("  Inserted medical record %d for patient %s\n", i+1, p.Name)
		}
	}

	fmt.Println("Seed data insertion complete.")
}
