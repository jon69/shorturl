// Модуль server представляет абстрацию сервера по обработке запросов.
package cookie

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"strings"

	uuid "github.com/satori/go.uuid"
)

func GetNewSignedCookie(secretKey []byte) (string, string, string) {
	// создаем новый идентификатор
	myuuid := uuid.NewV4()
	log.Println("new UUID is: ", myuuid.String())

	name := "uid"
	val := myuuid.String()

	mac := hmac.New(sha256.New, secretKey)
	mac.Write([]byte(name))
	mac.Write([]byte(val))
	signature := mac.Sum(nil)
	signatureHEX := hex.EncodeToString(signature)
	// Prepend the cookie value with the HMAC signature.
	val = signatureHEX + "-" + val
	return name, val, myuuid.String()
}

func ValidateCookie(secretKey []byte, name string, val string) (bool, string) {
	// A SHA256 HMAC signature has a fixed length of 32 bytes. To avoid a potential
	// 'index out of range' panic in the next step, we need to check sure that the
	// length of the signed cookie value is at least this long. We'll use the
	// sha256.Size constant here, rather than 32, just because it makes our code
	// a bit more understandable at a glance.
	signedValue := val
	if len(signedValue) < 4 {
		return false, ""
	}

	i := strings.Index(signedValue, "-")
	if i == -1 {
		log.Println("not found: - ")
		return false, ""
	}

	log.Println("i=", i)

	// Split apart the signature and original cookie value.
	signatureHEX := signedValue[:i]
	value := signedValue[i+1:]

	log.Println("signature=", signatureHEX)
	log.Println("value=", value)

	signature, err := hex.DecodeString(signatureHEX)
	if err != nil {
		log.Println("error to decode hex signature")
		return false, ""
	}

	// Recalculate the HMAC signature of the cookie name and original value.
	mac := hmac.New(sha256.New, secretKey)
	mac.Write([]byte(name))
	mac.Write([]byte(value))
	expectedSignature := mac.Sum(nil)

	// Check that the recalculated signature matches the signature we received
	// in the cookie. If they match, we can be confident that the cookie name
	// and value haven't been edited by the client.
	if !hmac.Equal(signature, expectedSignature) {
		log.Println("cookie not equal")
		return false, ""
	}
	log.Println("cookie equal")
	// Return the original cookie value.
	return true, value
}
