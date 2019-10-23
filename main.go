package main

import (
	"fmt"
	"os"
	"time"

	"gitlab.com/dj_arbz/traextor/acme"
	"gitlab.com/dj_arbz/traextor/internal"
)

func main() {

	// Load config from environmental variables
	acmeFile := internal.GetEnv("ACME_FILE", "/acme.json")
	outputDir := internal.GetEnv("OUTPUT_DIR", "/certificates")

	// Create certificate output dir
	if err := internal.CreateDir(outputDir); err != nil {
		internal.Log(fmt.Sprintf("Failed to create output dir %s: %v", outputDir, err))
		os.Exit(1)
	}

	if internal.GetEnv("BUILD_TEST", "") != "" {
		internal.Log("Dockerfile is running.")
		internal.Log("ACME is: " + acmeFile)
		internal.Log("Output directory is: " + outputDir)
		os.Exit(0)
	}

	// Check that the given acme.json file exists
	for !internal.CheckFileExists(acmeFile) {
		internal.Log(acmeFile + " does not exist!")
		// sleep for 30 seconds
		time.Sleep(30 * 1000 * time.Millisecond)
	}
	internal.Log(acmeFile + " found!")
	internal.Log("Your certificates will be exported to " + outputDir)

	// Create a new ACME store
	ACME := acme.New()

	if err := ACME.LoadFromFile(acmeFile); err != nil {
		internal.Log(fmt.Sprintf("Failed to load %s: %v", acmeFile, err))
		os.Exit(1)
	}

	if err := ACME.Generate(outputDir); err != nil {
		internal.Log(fmt.Sprintf("Failed to generate certificates: %v", err))
		os.Exit(1)
	}

	ACME.Watch(acmeFile, outputDir)

	internal.Log("Done?")
}
