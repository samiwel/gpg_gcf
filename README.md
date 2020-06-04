# Example: Google Cloud Function to decrypt a GPG encrypted file

This code has been adapted from [an symmetric key decryption example](https://github.com/salrashid123/gpg_gcf) to decrypt files encrypted with a public key.

This Cloud Function uses Go's `openpgp` library to decrypt a file that was encrypted offline using GPG.
The example assumes that the encrypted armored private key file and passphrase file are in a Google Cloud Storage bucket.

When the Decrypter function is invoked by a storage object finalize trigger, it decrypts the file using the decrypted private key and streams the decrypted file contents to a destination cloud storage bucket.

For convenience, a [standalone](./cmd/main.go) application has been created which uses the [Go Functions Framework](https://github.com/GoogleCloudPlatform/functions-framework-go) so that the function can be tested and invoked locally by issuing a POST HTTP request with the event payload in JSON.

## How to deploy

```
gcloud functions deploy Decrypter \
    --entry-point=Decrypter \
    --runtime go111 \
    --trigger-resource=${BUCKET_SRC} \
    --set-env-vars=BUCKET_DEST=${BUCKET_DEST},BUCKET_KEYS=${BUCKET_KEYS},PASSPHRASE_FILENAME=${PASSPHRASE_FILENAME} \
    --trigger-event=google.storage.object.finalize \
    --project=${GOOGLE_PROJECT_ID} \
    --timeout=540s \
    --memory=256MB
```

## Environment Variables

> An assumption has been made here that the armored private key and the passphrase file are in the same bucket. In practice, you would probably want to store these separately or use a service such as Secret Manager to store the passphrase file.

| Name        | Description           |
| ------------- |-------------|
| BUCKET_KEYS     | Name of GCS bucket containing the armored private key and passphrase file. |
| PASSPHRASE_FILENAME      | The name of the passphrase file.      |
| BUCKET_DEST | Name of bucket where decrypted files will be written to.      |