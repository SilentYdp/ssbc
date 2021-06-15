package util

import (
	"bytes"
	"compress/gzip"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"

	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/cloudflare/cfssl/log"
	"github.com/pkg/errors"
)

func ValidateAndReturnAbsConf(configFilePath, homeDir, cmdName string) (string, string, error) {
	var err error
	var homeDirSet bool
	var configFileSet bool

	defaultConfig := GetDefaultConfigFile(cmdName) // Get the default configuration

	if configFilePath == "" {
		configFilePath = defaultConfig // If no config file path specified, use the default configuration file
	} else {
		configFileSet = true
	}

	if homeDir == "" {
		homeDir = filepath.Dir(defaultConfig) // If no home directory specified, use the default directory
	} else {
		homeDirSet = true
	}

	// Make the home directory absolute
	homeDir, err = filepath.Abs(homeDir)
	if err != nil {
		return "", "", errors.Wrap(err, "Failed to get full path of config file")
	}
	homeDir = strings.TrimRight(homeDir, "/")

	if configFileSet && homeDirSet {
		log.Warning("Using both --config and --home CLI flags; --config will take precedence")
	}

	if configFileSet {
		configFilePath, err = filepath.Abs(configFilePath)
		if err != nil {
			return "", "", errors.Wrap(err, "Failed to get full path of configuration file")
		}
		return configFilePath, filepath.Dir(configFilePath), nil
	}

	configFile := filepath.Join(homeDir, filepath.Base(defaultConfig)) // Join specified home directory with default config file name
	return configFile, homeDir, nil
}

func GetDefaultConfigFile(cmdName string) string {

	if cmdName == "SSBC-server" {
		var fname = fmt.Sprintf("%s-config.yaml", cmdName)
		// First check home env variables
		home := "."
		return path.Join(home, fname)
	}

	var fname = fmt.Sprintf("%s-config.yaml", cmdName)
	return path.Join(os.Getenv("HOME"), ".SSBC-client", fname)
}

func FileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func Marshal(from interface{}, what string) ([]byte, error) {
	buf, err := json.Marshal(from)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to marshal %s", what)
	}
	return buf, nil
}

func Unmarshal(from []byte, to interface{}, what string) error {
	err := json.Unmarshal(from, to)
	if err != nil {
		return errors.Wrapf(err, "Failed to unmarshal %s", what)
	}
	return nil
}

//压缩
func Compress(in []byte) []byte  {
	//hash:="dsdadfafdff121e3edeeejfehu4ru4fneffferf"
	//buff:=make([]byte,0)
	//for i:=0;i<10000;i++{
	//	buff=append(buff,[]byte(hash)...)
	//}
	fmt.Println("origin size",len(in))
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	defer w.Close()
	w.Write(in)
	w.Flush()
	fmt.Println("gzip size:", len(b.Bytes()))
	return b.Bytes()


}

//解压
func DeCompress(in []byte)[]byte  {
	var b bytes.Buffer
	b.Write(in)
	r, _ := gzip.NewReader(&b)
	defer r.Close()
	undatas, _ := ioutil.ReadAll(r)
	fmt.Println("ungzip size:", len(undatas))
	return undatas
}


//数字签名
func RsaSignWithSha256(data []byte, keyBytes []byte) []byte {
	h := sha256.New()
	h.Write(data)
	hashed := h.Sum(nil)
	block, _ := pem.Decode(keyBytes)
	if block == nil {
		panic(errors.New("private key error"))
	}
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		log.Info("ParsePKCS8PrivateKey err", err)
		panic(err)
	}

	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hashed)
	if err != nil {
		fmt.Printf("Error from signing: %s\n", err)
		panic(err)
	}

	return signature
}

//签名验证
func RsaVerySignWithSha256(data, signData, keyBytes []byte) bool {
	block, _ := pem.Decode(keyBytes)
	if block == nil {
		panic(errors.New("public key error"))
	}
	pubKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		panic(err)
	}

	hashed := sha256.Sum256(data)
	err = rsa.VerifyPKCS1v15(pubKey.(*rsa.PublicKey), crypto.SHA256, hashed[:], signData)
	if err != nil {
		//panic(err)
		log.Info("验签不通过！")
		return false
	}
	return true
}
