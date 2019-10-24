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

// TraefikStore is the universal type of a Traefik interface
type TraefikStore interface {
	LoadFromFile(file string) error
	LoadJSON(input io.Reader) error
	Generate(outDir string) error
	Watch(file string, outDir string)
}

// New will give you a blank Traefik store
func New(version string) TraefikStore {
	internal.Log("")
	switch version {
	case "1":
		return new(Acme)
	case "2":
		return new(Traefik)
	default:
		return new(Traefik)
	}
}

// Traefik stores the raw JSON data and extracted data
type Traefik struct {
	CertStores map[string]*json.RawMessage
	CertStore  map[string]Acme
}

// LoadFromFile will import the provided JSON datafile
func (t *Traefik) LoadFromFile(file string) error {
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
			internal.Log("Error closing file %s: %v", file, err)
		}
	}()

	return t.LoadJSON(jsonFile)
}

// LoadJSON will populate the ACME store from a JSON reader
func (t *Traefik) LoadJSON(input io.Reader) error {
	byteValue, err := ioutil.ReadAll(input)
	if err != nil {
		return err
	}

	// Init Maps
	t.CertStores = make(map[string]*json.RawMessage)
	t.CertStore = make(map[string]Acme)

	// Unmarshal JSON CertStores
	if err := json.Unmarshal(byteValue, t.CertStores); err != nil {
		return err
	}

	// Unmarshal JSON Stores
	for certStore, rawJSON := range t.CertStores {
		if err := json.Unmarshal([]byte(*rawJSON), t.CertStore[certStore]); err != nil {
			return err
		}
	}

	internal.Log("Loaded ACME store!")
	return nil
}

// Generate will export all certificates in all cert stores to the specified directory
func (t *Traefik) Generate(outDir string) error {
	for certStoreName, certStoreData := range t.CertStore {
		if err := certStoreData.Generate(fmt.Sprintf("%s%c%s", outDir, os.PathSeparator, certStoreName)); err != nil {
			return err
		}
	}

	return nil
}

// Watch will reload the given file when it changes and reexport certificates
func (t *Traefik) Watch(file string, outDir string) {
	// creates a new file watcher
	watch(file, t.Generate, outDir)
}

// Acme is the raw V1 JSON data store
type Acme struct {
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
	URI  string                      `json:"uri,omitempty"`
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
func (a *Acme) LoadFromFile(file string) error {
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
func (a *Acme) LoadJSON(input io.Reader) error {
	byteValue, err := ioutil.ReadAll(input)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(byteValue, a); err != nil {
		return err
	}
	internal.Log("Loaded ACME store!")
	return nil
}

// Generate will process every certificate in the store and output to file
func (a *Acme) Generate(outDir string) error {
	internal.Log(fmt.Sprintf("Generating certificates for %d domains", len(a.Certificates)))
	for _, cert := range a.Certificates {
		if err := cert.generate(outDir); err != nil {
			return err
		}
	}

	return nil
}

// Watch will reload the given file when it changes and reexport certificates
func (a *Acme) Watch(file string, outDir string) {
	// creates a new file watcher
	watch(file, a.Generate, outDir)
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

func watch(file string, generateFunc func(string) error, outDir string) {

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
				if err := generateFunc(outDir); err != nil {
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

	internal.Log("Watching: " + file)

	<-done
}
