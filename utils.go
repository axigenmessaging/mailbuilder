package mailbuilder

import (
	"encoding/base64"
	"bytes"
	"mime/quotedprintable"
	"strings"
	"io"
	"io/ioutil"
	"crypto/rand"
	"fmt"
)

/**
 * Try to encode bytes using mime encoding
 */
func EncodeByContentEncoding(body []byte, encoding string) []byte {
	switch encoding {
	case "base64":
		b := make([]byte, base64.StdEncoding.EncodedLen(len(body)))
		base64.StdEncoding.Encode(b, body)
		return ByteBreakLines(b, 76, "\n")
	case "quoted-printable":
		b := bytes.NewBuffer(body)
		qpWriter := quotedprintable.NewWriter(b)
		qpWriter.Write(body)
		qpWriter.Close()
		return b.Bytes()
	default:
		return body
	}
}


/**
 * Try to decode mime encoded bytes
 */
func DecodeByContentEncoding(body []byte, encoding string) ([]byte, bool, error) {
	switch encoding {
	case "base64":
		//fmt.Println("-----------", string(body), "\r\n-------------")
		data, err := base64.StdEncoding.DecodeString(strings.Trim(string(body), "\r\n\t"))
		if err != nil {
			return nil, false, err
		}
		return data, true, nil
	case "quoted-printable":
		data, err := ioutil.ReadAll(quotedprintable.NewReader(strings.NewReader(string(body))))
		if err != nil {
			return nil, false, err
		}
		return data, true, nil
	default:
		return body, false, nil
	}
}

/**
 * generate a random boundary
 */

func RandomBoundary() string {
	var buf [30]byte
	_, err := io.ReadFull(rand.Reader, buf[:])
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%x", buf[:])
}


// break line with a separator
func ByteBreakLines(data []byte, charsNo int, lineSeparator string) []byte {
	result := StringBreakLines(string(data), charsNo, lineSeparator)
	return []byte(result)
}


// break line with a separator
func StringBreakLines(data string, charsNo int, lineSeparator string) string {

	var b strings.Builder

	startIdx := 0
	stopIdx := 0
	maxLength := len(data)
	stop := false

	for !stop {
		if startIdx > 0 {
			b.Grow(len(lineSeparator))
			b.WriteString(lineSeparator)
		}

		stopIdx = startIdx + charsNo
		if stopIdx > maxLength {
			stopIdx = maxLength
		}
		b.Grow(stopIdx - startIdx)
		b.WriteString(data[startIdx:stopIdx])

		startIdx += charsNo
		if startIdx >= maxLength {
			stop = true
		}
	}

	return b.String()
}


/**
 * for debugging purposes, return the message structure decoded
 */
func DebugMessageStructure(m *Message, prefix string) string {
	result := ""
	result += fmt.Sprintf(prefix+"IDX: %s\r\n", m.Idx)
	result += fmt.Sprintf(prefix+"Content-Type: %s\r\n", m.Header.Get("Content-Type"))
	result += fmt.Sprintf(prefix+"Is Multipart: %t\r\n", m.IsMultipart())
	result += fmt.Sprintf(prefix+"Is RFC822: %t\r\n", m.IsRfc822())

	if m.IsRfc822() {
		prefix += "     "
		//fmt.Println(prefix, "Unpack Body: ", m.IsMultipart())
		result += DebugMessageStructure(m.BodyMessage, prefix)
	}

	result += fmt.Sprintf(prefix+"Parts: %d\r\n", len(m.Parts))

	if len(m.Parts) > 0 {
		prefix += "     "
		for _, p := range m.Parts {
			result += DebugMessageStructure(p, prefix)
		}
	}

	return result
}