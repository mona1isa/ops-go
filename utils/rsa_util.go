package utils

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
)

var PrivateKey = `
-----BEGIN RSA PRIVATE KEY-----
MIIEogIBAAKCAQEA6mDxtXnT/kMPXfYLJaO7vtuuIMwixTSbmRBBShR8WuivXeYz
J5SirEARd6ivzJUXhkxJGGGU+pOmVDahGdY2qf/Usvux9/t0yyqzzGBpzR1RMiTm
eKqAh18+JZnpWakyJEub26tr9KIRaiFljCrDUkdhzOLoMAyXkaBFC7E4kQ0dLF8g
LfPDg2bM9AxGwSNqaVN7DNCajpvK6kXa/iwQf8ryiy2sKC/jcjBNnmF7FQ7LXy8w
4TxeCu1eTA6h5OGpMw6jhPLDrlidMo7DX07HirKI26el66/S00/KC8MtAZbIY52u
RDeC4ghxCbmIlT9tvMiwNzxR8GBbZCCU72GgswIDAQABAoIBAET4aTh7lNA2QK/o
Rwh5Abcc40VgfPzmScGFoIjhtWR1I6Hwq24C1cn1go5hU/ZSi10oVrw3fwUr7N7M
QqUdPfHRyHAxFAKC+zAMLWO/nXLQJUQpyq6IkhEEDIA5JguN+CTpIQDIFZkkFhbS
pBTWwqqUOen2fdgh5HpknNzfdmNsVbvFpvnHJhA3uXB7M+o1PGH/zZl6AgzMEFBS
aIFkbNGsZx+P4s9vcE4UCVFNdXiUiccxdUfNq4dSRLdtnvf+G8ib/4ewor9TLCLX
y86PxXTcap7hPvv8AC2zRKewK8nY/0DsXCS8ovcTA3VFzf15nrw9e9TtUxyyDv3B
+puJXxUCgYEA68dPAU8yzZz6aQL4g1dLZnuVBzKyXuNYequSdYT9iV4eddHBiJMg
0H5e5N1xshmnkGsxDVmfEKUejGMUkBNJxwzDK4J364aQ5RFrhfSwxAEmj/AeBwO2
qlckozyYsuqBK9BfYc89oD5NR5WVb2Yy0jFuq8tuNw+Zeuvk1ULL3GcCgYEA/nrm
jJAFzzUfWEBDPyFgubCB67ieghdhRZtueOcgXIiDSxleTWRMov2i6ujXiT0cSuRy
CwI/v3Mtf3Kvjl6GUWQs7EauPICcwElHoRMXti4VGhhVTVwLE4ChIM9T2KmAfosV
t+4t54B0hB6UGk7yHAVP8PE/r3MR3fhL/dSbadUCgYAIYP/cwwzCI9b+Tl24hSyn
yrKEG/gcySWGznwY8w3ziMW6WCbxjJD499S1e20j8Cd1SWnn2Ix/ke6g/JBpglX3
3es9q5hJZXHWwiS5EPYLMSNGsDjQ9P/T0974chnXGeBXR0NsfWnqPOyQI6+40r/x
mlIdhtA24rYImUN7lLEb9wKBgBChgY2wH+ERzLGcyYhHqyWXhnYcQ6em1YGSDd8y
46eIeGQhDUurgWKphsspWmSqrL2sPlO/2uCtK00H9rcsMEUDcfgjCmID2bqrT1YU
hFkwm8pvyqtal5K3tlAJnKYtNauPdWTm2PMnLvYvdWhevm3cXwQVEB9sOr+x6W12
Ro3dAoGAB4tUqCUaVEboXT48S88Y5ryDtIv2yo6x0sbN648ZHS7N4FU3YddyZcwC
3tFQD0tdQnfwLrDCQ59vsUt5zx1W85f5SgH0XLd54m4N/I0+jA7707cBCLNFhTk9
junOCJHGS3ASKt47XdnoCBTZMOTn/08fC7Ndtd1FPdnIFpY2kLg=
-----END RSA PRIVATE KEY-----
`

// GenerateKeyPair generates a new RSA key pair.
func GenerateKeyPair(bits int) (*rsa.PrivateKey, *rsa.PublicKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, nil, err
	}
	return privateKey, &privateKey.PublicKey, nil
}

// Encrypt encrypts data using RSA public key.
// Encrypt uses RSA public key to encrypt the given data
// It utilizes PKCS#1 v1.5 padding scheme for encryption
//
// Parameters:
//
//	publicKey: The RSA public key to use for encryption
//	data: The plaintext data to be encrypted
//
// Returns:
//
//	[]byte: The encrypted ciphertext
//	error: Any error that occurred during encryption
func Encrypt(publicKey *rsa.PublicKey, data []byte) ([]byte, error) {
	// Call the standard library function to encrypt data with PKCS#1 v1.5 padding
	return rsa.EncryptPKCS1v15(rand.Reader, publicKey, data)
}

// Decrypt decrypts data using RSA private key.
// Decrypt decrypts the ciphertext using RSA PKCS#1 v1.5 decryption with the given private key.
//
// Parameters:
//
//	privateKey: The RSA private key to use for decryption
//	ciphertext: The encrypted data to decrypt
//
// Returns:
//
//	[]byte: The decrypted plaintext
//	error: An error if the decryption fails
func Decrypt(privateKey *rsa.PrivateKey, ciphertext []byte) ([]byte, error) {
	return rsa.DecryptPKCS1v15(rand.Reader, privateKey, ciphertext)
}

// ExportPrivateKeyToPEM exports private key to PEM format.
// ExportPrivateKeyToPEM converts an RSA private key to PEM format string representation
// Parameters:
//
//	privateKey: The RSA private key to be exported
//
// Returns:
//
//	string: The PEM encoded string representation of the private key
//	error: Any error that might occur during the conversion process
func ExportPrivateKeyToPEM(privateKey *rsa.PrivateKey) (string, error) {
	// Marshal the private key into PKCS#1 format
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	// Encode the private key bytes into PEM format
	privateKeyPEM := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY", // Specify the PEM block type for RSA private key
			Bytes: privateKeyBytes,   // The marshaled private key bytes
		},
	)
	// Convert the PEM encoded bytes to string and return
	return string(privateKeyPEM), nil
}

// ExportPublicKeyToPEM exports public key to PEM format.
// ExportPublicKeyToPEM converts an RSA public key to PEM format
// Parameters:
//
//	publicKey - The RSA public key to be converted
//
// Returns:
//
//	string - The PEM formatted public key
//	error - Any error that occurred during the conversion
func ExportPublicKeyToPEM(publicKey *rsa.PublicKey) (string, error) {
	// Marshal the public key into PKIX format
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return "", err
	}
	// Encode the marshaled key into PEM format
	publicKeyPEM := pem.EncodeToMemory(
		&pem.Block{
			Type:  "PUBLIC KEY",   // PEM block type for public key
			Bytes: publicKeyBytes, // The marshaled public key bytes
		},
	)
	// Convert the PEM bytes to string and return
	return string(publicKeyPEM), nil
}

// ParsePrivateKeyFromPEM parses private key from PEM format.
// ParsePrivateKeyFromPEM parses a PEM-encoded private key and returns an RSA private key.
// It takes a string containing PEM-encoded data and returns either the parsed private key
// or an error if the parsing fails.
//
// Parameters:
//
//	pemData: string containing PEM-encoded private key data
//
// Returns:
//
//	*rsa.PrivateKey: the parsed RSA private key
//	error: any error encountered during parsing (e.g., invalid PEM format, invalid key format)
func ParsePrivateKeyFromPEM(pemData string) (*rsa.PrivateKey, error) {
	// Decode the PEM-encoded data to extract the PEM block
	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		// Return error if no valid PEM block is found
		return nil, errors.New("failed to parse PEM block containing the key")
	}
	// Parse the PKCS1 private key from the PEM block bytes
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		// Return error if the key cannot be parsed as PKCS1 format
		return nil, err
	}
	// Return the successfully parsed private key
	return privateKey, nil
}

// GetPrivateKey returns the *rsa.PrivateKey from the global privateKey string.
// GetPrivateKey retrieves a private key by parsing it from PEM format
// It returns the parsed RSA private key and an error if any occurs during parsing
func GetPrivateKey() (*rsa.PrivateKey, error) {
	// Parse the private key from the PEM formatted string and return it
	// along with any error that might occur during parsing
	return ParsePrivateKeyFromPEM(PrivateKey)
}

// ParsePublicKeyFromPEM parses public key from PEM format.
// ParsePublicKeyFromPEM parses a PEM-encoded public key string and returns an RSA public key.
// It takes a PEM-encoded string as input and returns the parsed RSA public key or an error.
//
// Parameters:
//
//	pemData: string containing the PEM-encoded public key
//
// Returns:
//
//	*rsa.PublicKey: the parsed RSA public key
//	error: any error encountered during parsing (e.g., invalid PEM format, non-RSA key)
func ParsePublicKeyFromPEM(pemData string) (*rsa.PublicKey, error) {
	// Decode the PEM-encoded data into a PEM block
	// The pem.Decode function returns a block and any remaining data, which we ignore with _
	block, _ := pem.Decode([]byte(pemData))
	// Check if the PEM block was successfully decoded
	if block == nil {
		return nil, errors.New("failed to parse PEM block containing the key")
	}
	// Parse the PKIX-encoded public key from the PEM block bytes
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	// Type assertion to ensure the parsed key is an RSA public key
	publicKey, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("not an RSA public key")
	}
	// Return the successfully parsed RSA public key
	return publicKey, nil
}
