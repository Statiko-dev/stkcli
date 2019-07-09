/*
Copyright © 2019 Alessandro Segala (@ItalyPaleAle)

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program. If not, see <http://www.gnu.org/licenses/>.
*/

package cmd

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/spf13/cobra"
	pkcs12 "software.sslmate.com/src/go-pkcs12"

	"smpcli/utils"
)

func init() {
	var (
		name        string
		certificate string
		key         string
		dhparams    string
	)

	// This function loads the certificate
	var loadCertificate = func(file string) (*x509.Certificate, error) {
		// Load certificate from disk
		dataPEM, err := ioutil.ReadFile(file)
		if err != nil {
			return nil, err
		}

		// Get the certificate
		block, _ := pem.Decode(dataPEM)
		if block == nil || block.Type != "CERTIFICATE" {
			return nil, errors.New("Cannot decode PEM block containing certificate")
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, err
		}

		return cert, nil
	}

	// This function loads the private key
	var loadPrivateKey = func(file string) (*rsa.PrivateKey, error) {
		// Load key from disk
		dataPEM, err := ioutil.ReadFile(file)
		if err != nil {
			return nil, err
		}

		// Parse the key
		block, _ := pem.Decode(dataPEM)
		if block == nil || block.Type != "RSA PRIVATE KEY" {
			return nil, errors.New("Cannot decode PEM block containing private key")
		}
		prv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}

		return prv, nil
	}

	// This function creates a PCKS12-encoded file (PFX) with the certificates
	var createPFX = func() []byte {
		// Load the certificate and key
		crt, err := loadCertificate(certificate)
		if err != nil {
			fmt.Println("[Fatal error]\nCannot load certificate:", err)
			return nil
		}
		prv, err := loadPrivateKey(key)
		if err != nil {
			fmt.Println("[Fatal error]\nCannot load private key:", err)
			return nil
		}

		// Crete the PCKS12 bag
		pcksData, err := pkcs12.Encode(rand.Reader, prv, crt, nil, "")
		if err != nil {
			fmt.Println("[Fatal error]\nCannot create PKCS12 bag:", err)
			return nil
		}

		return pcksData
	}

	// This function returns true if the file exists and it's a regular file
	var checkFile = func(path string) bool {
		// Check if the path exists
		isFile, err := utils.IsRegularFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Println("[Error]\nFile not found:", path)
				return false
			}
			fmt.Println("[Fatal error]\nError while reading filesystem:", err)
			return false
		}
		if !isFile {
			fmt.Println("[Error]\nFile not found:", path)
			return false
		}
		return true
	}

	c := &cobra.Command{
		Use:   "upload-certificate",
		Short: "Upload a certificate",
		Long:  ``,
		Run: func(cmd *cobra.Command, args []string) {
			// Check if all files exist
			if !checkFile(certificate) {
				return
			}
			if !checkFile(key) {
				return
			}
			if !checkFile(dhparams) {
				return
			}

			// Convert the certificate and key to PCKS12
			result := createPFX()
			if result == nil {
				return
			}

		},
	}
	rootCmd.AddCommand(c)

	// Flags
	c.Flags().StringVarP(&name, "name", "n", "", "Certificate name (required)")
	c.MarkFlagRequired("name")
	c.Flags().StringVarP(&certificate, "certificate", "c", "", "Certificate file (required)")
	c.MarkFlagRequired("certificate")
	c.Flags().StringVarP(&key, "key", "k", "", "Private key (required)")
	c.MarkFlagRequired("key")
	c.Flags().StringVarP(&dhparams, "dhparams", "d", "", "DH Parameters file (required)")
	c.MarkFlagRequired("dhparams")
}