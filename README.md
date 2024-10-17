# lockboxkms

A simple web interface for encrypting text using Google Cloud KMS.

## Overview

`LockboxKMS` is a web application that provides a user-friendly interface to encrypt text data using Google Cloud Key Management Service (KMS). It supports multiple encryption keys and offers flexible key management, ensuring one-way data protection by focusing solely on encryption.

This application provides an extremely simple web interface for encrypting data using Google Cloud KMS providing the options for selecting a key in the KMS key ring and encrypting the data using that key.

The encrypted data is returned to the user base64 encoded, and can be decrypted using the same key in the KMS key ring, but this interface intentionally does not provide a decryption option.  The intent here is to provide a simple way to encrypt data using KMS, and then store the encrypted data somewhere so that a separate process can later use the same key to decrypt the data.

## Configuration

The application is configurable through environment variables. Below are the available configurations:

- `GCP_PROJECT`: Your Google Cloud project ID. (required, application will not start without it)
- `KMS_LOCATION`: The location of your KMS resources (default: us).
- `KMS_KEY_RING`: The name of the KMS key ring to use (default: lockboxkms).
- `PORT`: The port on which the server listens (default: 8080).
