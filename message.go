package mailbuilder

import (
	"net/textproto"
	"strings"
	"bytes"
	"bufio"
	//"fmt"
)

type Message struct {
	// mail header
	Header            textproto.MIMEHeader
	HeaderIsChanged bool

	// original raw header extracted with decomposer
	RawOriginalHeader []byte

	// original headers orders
	HeaderOrder       []string

	// simple message body
	Body              []byte

	// message parts if the message is multipart
	Parts             []*Message

	// if the message is rfc822 the body is itself a message
	BodyMessage       *Message

	// boundary used for multiparts
	Boundary          string
	Idx               string

	// specify if the message body is mime decoded
	IsDecoded         bool

	// rfc822 depth
	rfc822Depth       int

	// the parent of the Message/Part
	Parent       *Message
}

// check if the message is multipart
func (c *Message) IsMultipart() bool {
	if len(c.Parts) > 0 {
		return true
	}
	return false
}

// append a part to message
func (c *Message) AddPart(p *Message) {
	p.Parent = c
	c.Parts = append(c.Parts, p)
}

// check if the message is RFC822
func (c *Message) IsRfc822() bool {
	return  c.BodyMessage != nil
}


// set the original header when decompose
func (c *Message) SetOriginalHeaderOrder(body []byte) {
	bReader := bytes.NewReader(body)
	r := bufio.NewReader(bReader)

	c.HeaderOrder = make([]string, 0)

	for {
		lineByte, _, err := r.ReadLine()
		lineString := string(lineByte)

		if err != nil {
			break
		}
		if len(lineString) > 0 {
			lineParts := strings.Split(lineString, ":")
			if !strings.HasPrefix(lineParts[0], " ") && !strings.HasPrefix(lineParts[0], "\t") {
				c.HeaderOrder = append(c.HeaderOrder, strings.Trim(lineParts[0], ""))
			}
		} else {
			//fmt.Println("BREAK EMPTY LINE", len(lineString))
			break
		}

		if len(c.RawOriginalHeader) > 0 {
			c.RawOriginalHeader = append(c.RawOriginalHeader, []byte("\n")...)
		}
		c.RawOriginalHeader = append(c.RawOriginalHeader, lineByte...)
	}
}

// copy into c Message the properties from m Message
func (c *Message) Merge(m *Message) {
	// keep the original headers, and rewrite only the new ones
	for key, val := range m.Header {
		if val[0] != "" {
			c.Header.Set(key, val[0])
		} else {
			c.Header.Del(key)
		}
	}

	c.BodyMessage  = m.BodyMessage
	c.Body  = m.Body
	c.Boundary  = m.Boundary
	c.Parts = m.Parts
	c.HeaderIsChanged = true
}