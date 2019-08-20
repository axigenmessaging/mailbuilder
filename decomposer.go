package mailbuilder

import (
	"bytes"
	"io"
	"io/ioutil"
	"strconv"
	"strings"
	"net/textproto"
	"os"
	"mime"
	"net/mail"
	"bufio"
	"aximailbuilder/mail-multipart"
	"aximailbuilder/mail-textproto"
)

// read an email
func ReadMessage(r io.Reader) (msg *mail.Message, rawOriginalHeader []byte, err error) {
	tp := mailtextproto.NewReader(bufio.NewReader(r))

	hdr, rawOriginalHeader, err := tp.ReadMIMEHeader()
	if err != nil {
		return nil, rawOriginalHeader, err
	}

	return &mail.Message{
		Header: mail.Header(hdr),
		Body:   tp.R,
	}, rawOriginalHeader, nil
}

type MessageDecomposer struct {}

func NewMessageDecomposer() MessageDecomposer {
	return MessageDecomposer{}
}

// decompose a message in components: header, body, parts
func (d *MessageDecomposer) Decompose(rawMessage []byte, partIdx string) (result *Message, err error) {
	reader := bytes.NewReader(rawMessage)
	//msg, err := mail.ReadMessage(reader)
	msg, originalHeader, err := ReadMessage(reader)

	if err != nil {
		return nil, err
	}

	if msg != nil {
		result = &Message{}
		result.Idx = partIdx
		result.Header = textproto.MIMEHeader(msg.Header)
		result.rfc822Depth = 0
		//result.SetOriginalHeaderOrder(rawMessage)
		result.SetOriginalHeaderOrder(originalHeader)

		err := d.ReadParts(result, msg.Body)
		if err != nil {
			return nil, err
		}
		return result, nil
	}
	return nil, err
}


// decompose an eml in parts
func (d *MessageDecomposer) DecomposeFile(file string) (*Message, error) {
	if _, err := os.Stat(file); err != nil {
		return nil, err
	}

	rawMessage, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	return d.Decompose(rawMessage, "")
}


// extract boundary if exists
func (d *MessageDecomposer) ExtractBoundary(header textproto.MIMEHeader) (string, error) {
	_, params, err := mime.ParseMediaType(header.Get("Content-Type"))
	if boundary, ok := params["boundary"]; ok {
		return boundary, nil
	}
	return "", err
}


// read message parts
func (d *MessageDecomposer) ReadParts(result *Message, bodyReader io.Reader) error {
	boundary, _ := d.ExtractBoundary(result.Header)

	if boundary != "" {
		// Multipart
		result.Boundary = boundary

		reader := mailmultipart.NewReader(bodyReader, result.Boundary)
		var idx int64 = 0
		for {
			idx += 1
			part, err := reader.NextPart()

			if err == io.EOF {
				return nil
			}
			if err != nil {
				return err
			}

			newPartEmail := &Message{}
			newPartEmail.Header = part.Header
			newPartEmail.RawOriginalHeader = part.RawOriginalHeader
			newPartEmail.Idx = result.Idx
			newPartEmail.rfc822Depth = result.rfc822Depth

			if newPartEmail.Idx != "" {
				newPartEmail.Idx += "-"
			}
			newPartEmail.Idx += strconv.FormatInt(idx, 10)

			err = d.ReadParts(newPartEmail, part)
			if err != nil {
				return err
			}

			result.Parts = append(result.Parts, newPartEmail)
		}
	} else {
		rawPartBody, err := ioutil.ReadAll(bodyReader)
		if err != nil {
			return err
		}

		decodedAsMessage := false

		if strings.HasPrefix(strings.Trim(result.Header.Get("Content-Type"), " \t"), "message/rfc822") && result.rfc822Depth < 5 {
			/**
			 * If we get an message/rfc822 part try to see if it contains
			 * an email; goes to max 5 message/rfc822 depth
			 */
			// Try to parse the body as a new Message
			decodedBody, isDecoded, err := DecodeByContentEncoding(rawPartBody, result.Header.Get("Content-Transfer-Encoding"))
			if err == nil {
				// Try to decode the part if is base64 or quoted-printable to be parsed as email
				newMessage, err := d.Decompose(decodedBody, result.Idx+"-0")
				if err == nil {
					newMessage.rfc822Depth = result.rfc822Depth + 1
					result.BodyMessage = newMessage

					// Mark the body was decoded so we encode it back when recompose the email
					result.IsDecoded = isDecoded

					decodedAsMessage = true
				}
			}
		}

		if !decodedAsMessage {
			// The part has no more parts
			result.Body = rawPartBody
		}
	}
	return nil
}