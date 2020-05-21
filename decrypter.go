package gcfgpg

import (
	"bytes"
	"cloud.google.com/go/storage"
	"context"
	"crypto"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/packet"
	"io"
	"log"
	"os"
)

type Event struct {
	Bucket string `json:"bucket"`
	Name   string `json:"name"`
}

var client *storage.Client

func init() {
	var err error

	client, err = storage.NewClient(context.Background())
	if err != nil {
		log.Fatalf("storage.NewClient: %v", err)
	}

	keysBucketName := os.Getenv("BUCKET_KEYS")
	if len(keysBucketName) == 0 {
		log.Fatal("Environment variable `BUCKET_KEYS` must be set and not empty.")
	}

	passphraseFileName := os.Getenv("PASSPHRASE_FILENAME")
	if len(passphraseFileName) == 0 {
		log.Fatal("Environment variable `PASSPHRASE_FILENAME` must be set and not empty.")
	}

	destination := os.Getenv("BUCKET_DEST")
	if len(destination) == 0 {
		log.Fatal("Environment variable `BUCKET_DEST` must be set and not empty.")
	}
}

func Decrypter(ctx context.Context, event Event) error {
	log.Printf("Processing %s in bucket %s", event.Name, event.Bucket)

	srcBucket := client.Bucket(event.Bucket)

	dstBucket := client.Bucket(os.Getenv("BUCKET_DEST"))

	gcsSrcObject := srcBucket.Object(event.Name)
	gcsSrcReader, err := gcsSrcObject.NewReader(ctx)
	if err != nil {
		log.Fatal("[Decrypter] Error: (%s) ", err)
	}
	defer gcsSrcReader.Close()

	gcsDstObject := dstBucket.Object(event.Name + ".dec")
	gcsDstWriter := gcsDstObject.NewWriter(ctx)

	keysBucket := client.Bucket(os.Getenv("BUCKET_KEYS"))

	keyring := keysBucket.Object("keyring.asc")
	keyringReader, err := keyring.NewReader(ctx)
	if err != nil {
		log.Fatalf("Error creating reader for private key object: %s", err)
	}

	passphraseObj := keysBucket.Object(os.Getenv("PASSPHRASE_FILENAME"))
	passphraseReader, err := passphraseObj.NewReader(ctx)
	if err != nil {
		log.Fatalf("Error creating reader for passphrase file object: %s", err)
	}

	entityList, err := openpgp.ReadArmoredKeyRing(keyringReader)
	if err != nil {
		log.Fatalf("%v", err)
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(passphraseReader)
	passphraseBytes := buf.Bytes()

	// The -1 here is to remove the \n character at the end of the file.
	// There might be a nicer way of doing this...
	passphrase := passphraseBytes[:len(passphraseBytes)-1]
	log.Println("Decrypting private key using passphrase")
	entityList[0].PrivateKey.Decrypt(passphrase)
	for _, subkey := range entityList[0].Subkeys {
		subkey.PrivateKey.Decrypt(passphrase)
	}
	log.Println("Finished decrypting private key using passphrase")

	packetConfig := packet.Config{
		DefaultHash:   crypto.SHA512,
		DefaultCipher: packet.CipherAES256,
		RSABits:       4096,
	}

	md, err := openpgp.ReadMessage(gcsSrcReader, entityList, nil, &packetConfig)
	if err != nil {
		log.Fatalf("Error reading message: %s", err)
	}

	n, err := io.Copy(gcsDstWriter, md.UnverifiedBody)
	if err != nil {
		log.Fatalf("Error reading encrypted file: %s", err)
	}
	log.Printf("Decrypted %d bytes", n)

	err = gcsDstWriter.Close()
	if err != nil {
		log.Fatal(err)
	}

	return nil
}
