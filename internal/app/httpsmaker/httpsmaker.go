// Модуль httpsmaker представляет класс для создания HTTPS.
package httpsmaker

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log"
	"math/big"
	"net"
	"os"
	"time"
)

// MakeHTTPS создаем сертификат и секретный ключ
func MakeHTTPS() (string, string, error) {
	// создаём шаблон сертификата
	cert := &x509.Certificate{
		// указываем уникальный номер сертификата
		SerialNumber: big.NewInt(1658),
		// заполняем базовую информацию о владельце сертификата
		Subject: pkix.Name{
			Organization: []string{"evgeniy.corporation"},
			Country:      []string{"RU"},
		},
		// разрешаем использование сертификата для 127.0.0.1 и ::1
		IPAddresses: []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		// сертификат верен, начиная со времени создания
		NotBefore: time.Now(),
		// время жизни сертификата — 10 лет
		NotAfter:     time.Now().AddDate(10, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		// устанавливаем использование ключа для цифровой подписи,
		// а также клиентской и серверной авторизации
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:    x509.KeyUsageDigitalSignature,
	}

	// создаём новый приватный RSA-ключ длиной 4096 бит
	// обратите внимание, что для генерации ключа и сертификата
	// используется rand.Reader в качестве источника случайных данных
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		log.Println("can not GenerateKey " + err.Error())
		return "", "", err
	}

	// создаём сертификат x.509
	certBytes, err := x509.CreateCertificate(rand.Reader, cert, cert, &privateKey.PublicKey, privateKey)
	if err != nil {
		log.Println("can not CreateCertificate " + err.Error())
		return "", "", err
	}

	// кодируем сертификат и ключ в формате PEM, который
	// используется для хранения и обмена криптографическими ключами
	var certPEM bytes.Buffer
	err = pem.Encode(&certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})

	if err != nil {
		log.Println("can not pem.Encode certPEM " + err.Error())
		return "", "", err
	}

	var privateKeyPEM bytes.Buffer
	err = pem.Encode(&privateKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	if err != nil {
		log.Println("can not pem.Encode privateKeyPEM " + err.Error())
		return "", "", err
	}

	myCertFile := "./mycert.pem"
	myKeyFile := "./mykey.pem"

	fileCERT, err := os.Create(myCertFile)
	if err != nil {
		log.Printf("can not Create file %s | %s", myCertFile, err.Error())
		return "", "", err
	}
	defer fileCERT.Close()

	_, err = certPEM.WriteTo(fileCERT)
	if err != nil {
		log.Printf("can not WriteTo file %s | %s", myCertFile, err.Error())
		return "", "", err
	}

	fileKey, err := os.Create(myKeyFile)
	if err != nil {
		log.Printf("can not Create file %s | %s", myKeyFile, err.Error())
		return "", "", err
	}
	defer fileKey.Close()

	_, err = privateKeyPEM.WriteTo(fileKey)
	if err != nil {
		log.Printf("can not WriteTo file %s | %s", myKeyFile, err.Error())
		return "", "", err
	}

	return myCertFile, myKeyFile, nil
}
