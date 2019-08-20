/**
 * create an email from an message object; the result is a bytes sequence
 */

package mailbuilder

import (
	"bytes"
	"net/textproto"
)

func NewMessageBuilder() MessageBuilder {
	return MessageBuilder{}
}


type MessageBuilder struct {
	newLine string
}

func (c *MessageBuilder) SetNewline(nl string) {
	c.newLine = nl
}

func (c *MessageBuilder) GetNewline() (string) {
	return c.newLine
}


/**
 * build the message from components
 *
 */
func(c *MessageBuilder) Build(m *Message) ([]byte) {

	buff := bytes.NewBuffer([]byte{})

	// write header
	buff.Write(c.BuildHeader(m))

	// write header & body separator
	buff.WriteString(c.GetNewline() + c.GetNewline())

	// write body
	body := c.BuildBody(m)
	if m.IsDecoded {
		/*
		 * The original message had the body encoded and the
		 * decomposer decoded it (only for message/rfc822 content type)
		 * to try to parse the parts
		 */
		body = EncodeByContentEncoding(body, m.Header.Get("Content-Transfer-Encoding"))
	}
	buff.Write(body)

	return buff.Bytes()
}


/**
 * create header trying to keep the same header order as the original
 */

func (c *MessageBuilder) BuildHeader(m *Message) ([]byte) {

	if len(m.RawOriginalHeader) > 0 && !m.HeaderIsChanged {
		return bytes.TrimRight(m.RawOriginalHeader, "\r\n")
	}

	buff := bytes.NewBuffer([]byte{})

	alreadyAdded := make(map[string]bool)
	if m.HeaderOrder != nil && len(m.HeaderOrder) > 0 {
		for _, headerCode := range m.HeaderOrder {
			if _, ok := m.Header[textproto.CanonicalMIMEHeaderKey(headerCode)]; ok {
				if buff.String() != "" {
					buff.WriteString(c.GetNewline())
				}
				buff.WriteString(headerCode + ": " + m.Header.Get(headerCode))
				alreadyAdded[textproto.CanonicalMIMEHeaderKey(headerCode)] = true
			}
		}
	}

	for key, _ := range m.Header {
		if _, ok := alreadyAdded[key]; ok {
			continue
		}
		if buff.String() != "" {
			buff.WriteString(c.GetNewline())
		}
		buff.WriteString(key + ": " + m.Header.Get(key))
	}

	return buff.Bytes()
}


/**
 * create message body
 */

func (c *MessageBuilder) BuildBody(m *Message) ([]byte) {
	buff := bytes.NewBuffer([]byte{})

	if m.IsRfc822() {
		buff.Write(c.Build(m.BodyMessage))
	} else if len(m.Body) > 0 {
		buff.Write(m.Body)
	}

	if m.IsMultipart() {
		// be sure we have a bondary set
		if m.Boundary == "" {
			m.Boundary = RandomBoundary()
		}

		for idx, part := range m.Parts {
			if idx > 0 {
				buff.WriteString(c.GetNewline())
			}
			// open boundary
			buff.WriteString(c.GetNewline()+"--"+m.Boundary+c.GetNewline())

			// build part message
			buff.Write(c.Build(part))
		}
		// close boundary
		buff.WriteString(c.GetNewline()+"--"+m.Boundary+"--"+c.GetNewline())

	}

	return buff.Bytes()
}

