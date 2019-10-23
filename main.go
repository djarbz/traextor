package main

import (
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
		internal.Log("Failed to create output dir %s: %v", outputDir, err)
		os.Exit(1)
	}

	if internal.GetEnv("BUILD_TEST", "") != "" {
		internal.Log("Dockerfile is running.")
		internal.Log("ACME is: %s", acmeFile)
		internal.Log("Output directory is: %s", outputDir)
		os.Exit(0)
	}

	// Check that the given acme.json file exists
	for !internal.CheckFileExists(acmeFile) {
		internal.Log("%s does not exist!", acmeFile)
		// sleep for 30 seconds
		time.Sleep(30 * 1000 * time.Millisecond)
	}
	internal.Log("%s found!", acmeFile)
	internal.Log("Your certificates will be exported to %s", outputDir)

	// Create a new ACME store
	ACME := acme.New(internal.GetEnv("TRAEFIK_VERSION", "2"))

	if err := ACME.LoadFromFile(acmeFile); err != nil {
		internal.Log("Failed to load %s: %v", acmeFile, err)
		os.Exit(1)
	}

	if err := ACME.Generate(outputDir); err != nil {
		internal.Log("Failed to generate certificates: %v", err)
		os.Exit(1)
	}

	ACME.Watch(acmeFile, outputDir)

	internal.Log("Done?")
}
