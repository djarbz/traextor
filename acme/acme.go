package acme

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/fsnotify/fsnotify"

	"gitlab.com/dj_arbz/traextor/internal"
)

// New will give you a blank ACME store
func New() *acme {
	return new(acme)
}

type acme struct {
	Account        acmeAccount        `json:"Account,omitempty"`
	Certificates   []acmeCertificate  `json:"Certificates,omitempty"`
	HTTPChallenges acmeHTTPChallenges `json:"HTTPChallenges,omitempty"`
}

type acmeAccount struct {
	Email        string                  `json:"Email,omitempty"`
	Registration acmeAccountRegistration `json:"Registration,omitempty"`
	PrivateKey   string                  `json:"PrivateKey,omitempty"`
}

type acmeAccountRegistration struct {
	Body acmeAccountRegistrationBody `json:"body,omitempty"`
	Uri  string                      `json:"uri,omitempty"`
}

type acmeAccountRegistrationBody struct {
	Status  string   `json:"status,omitempty"`
	Contact []string `json:"contact,omitempty"`
}

type acmeCertificate struct {
	Domain      acmeCertificateDomain `json:"Domain,omitempty"`
	Certificate string                `json:"Certificate,omitempty"`
	Key         string                `json:"Key,omitempty"`
}

type acmeCertificateDomain struct {
	Main string   `json:"Main,omitempty"`
	SANs []string `json:"SANs"`
}

type acmeHTTPChallenges struct {
}

// LoadFromFile will populate the ACME store from a JSON file
func (a *acme) LoadFromFile(file string) error {
	// Check file is accessible
	if !internal.CheckFileExists(file) {
		return fmt.Errorf("acme file does not exist: %s", file)
	}

	// Load file from disk
	jsonFile, err := os.Open(file)
	if err != nil {
		return err
	}
	defer func() {
		if err := jsonFile.Close(); err != nil {
			fmt.Printf("Error closing file %s: %v", file, err)
		}
	}()

	return a.LoadJSON(jsonFile)
}

// LoadJSON will populate the ACME store from a JSON reader
func (a *acme) LoadJSON(input io.Reader) error {
	byteValue, err := ioutil.ReadAll(input)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(byteValue, a); err != nil {
		return err
	}
	return nil
}

// Generate will process every certificate in the store and output to file
func (a *acme) Generate(outDir string) error {
	for _, cert := range a.Certificates {
		if err := cert.generate(outDir); err != nil {
			return err
		}
	}

	return nil
}

// Watch will reload the given file when it changes and reexport certificates
func (a *acme) Watch(file string, outDir string) {
	// creates a new file watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		internal.Log(fmt.Sprintf("Failed to create watcher: %v", err))
		os.Exit(1)
	}
	defer func() {
		err := watcher.Close()
		if err != nil {
			internal.Log(fmt.Sprintf("Failed to close watcher: %v", err))
		}
	}()

	//
	done := make(chan bool)

	go func() {
		for {
			select {
			// watch for events
			case event := <-watcher.Events:
				internal.Log(fmt.Sprintf("EVENT! %#v\n", event))
				if err := a.Generate(outDir); err != nil {
					internal.Log(fmt.Sprintf("Error generating certificates: %v", err))
				}
			// watch for errors
			case err := <-watcher.Errors:
				internal.Log(fmt.Sprintf("Error processing watcher event: %v", err))
			}
		}
	}()

	// out of the box fsnotify can watch a single file, or a single directory
	if err := watcher.Add(file); err != nil {
		internal.Log(fmt.Sprintf("Failed to add %s to watcher: %v", file, err))
		os.Exit(1)
	}

	<-done
}

func (ac *acmeCertificate) generate(outDir string) error {
	key, err := ac.getKeyBytes()
	if err != nil {
		return err
	}

	cert, err := ac.getCertBytes()
	if err != nil {
		return err
	}

	// Create certificates for SANs
	for _, san := range ac.Domain.SANs {
		internal.Log("SAN: " + san)
		if err := ac.Domain.writeCerts(san, cert, key, outDir); err != nil {
			return err
		}
	}

	// Save the certificate to disk
	internal.Log("Domain: " + ac.Domain.Main)
	return ac.Domain.writeCerts(ac.Domain.Main, cert, key, outDir)
}

func (ac *acmeCertificate) getKeyBytes() ([]byte, error) {
	return base64.StdEncoding.DecodeString(ac.Key)
}

func (ac *acmeCertificate) getCertBytes() ([]byte, error) {
	return base64.StdEncoding.DecodeString(ac.Certificate)
}

func (acd *acmeCertificateDomain) writeCerts(domain string, certificate []byte, key []byte, outDir string) error {
	if err := internal.WriteFile(fmt.Sprintf("%s%c%s.pem", outDir, os.PathSeparator, domain), certificate); err != nil {
		return err
	}
	return internal.WriteFile(fmt.Sprintf("%s%c%s.key", outDir, os.PathSeparator, domain), key)
}