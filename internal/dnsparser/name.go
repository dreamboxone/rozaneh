// SPDX-License-Identifier: MIT
// Adapted from WhiteDNS/CottenDNS at 7db7b73658c791cce62300d29647e4d369d6e1e4.
// Copyright (c) 2026 Amin Mahmoudi, NullRoute1970, tajirax.
// See licenses/CottenDNS-LICENSE.
package dnsparser

import (
	"encoding/binary"
	"errors"
	"strings"
)

var ErrInvalidName = errors.New("invalid DNS name")

const maxNameJumps = 10

func parseName(data []byte, offset int) (string, int, error) {
	dataLen := len(data)
	if offset >= dataLen {
		return "", offset, ErrInvalidName
	}

	var (
		jumped   bool
		jumps    int
		origNext = offset
		name     strings.Builder
		hasLabel bool
	)

	for {
		if offset >= dataLen {
			return "", origNext, ErrInvalidName
		}

		length := int(data[offset])
		if length == 0 {
			offset++
			if !jumped {
				origNext = offset
			}
			break
		}

		if length >= 192 { // 0xC0
			if offset+1 >= dataLen || jumps >= maxNameJumps {
				return "", origNext, ErrInvalidName
			}
			ptr := int(binary.BigEndian.Uint16(data[offset:offset+2]) & 0x3FFF)
			if ptr >= dataLen {
				return "", origNext, ErrInvalidName
			}
			if !jumped {
				origNext = offset + 2
				jumped = true
			}
			offset = ptr
			jumps++
			continue
		}

		if length > 63 {
			return "", origNext, ErrInvalidName
		}

		offset++
		end := offset + length
		if end > dataLen {
			return "", origNext, ErrInvalidName
		}

		if name.Len() == 0 {
			name.Grow(64)
		} else {
			name.WriteByte('.')
		}

		writeLowerASCIILabel(&name, data[offset:end])
		hasLabel = true
		offset = end
		if !jumped {
			origNext = offset
		}
	}

	if !hasLabel {
		return ".", origNext, nil
	}
	return name.String(), origNext, nil
}

func writeLowerASCIILabel(dst *strings.Builder, label []byte) {
	upperIndex := -1
	for i := range label {
		if label[i] >= 'A' && label[i] <= 'Z' {
			upperIndex = i
			break
		}
	}

	if upperIndex == -1 {
		dst.Write(label)
		return
	}

	dst.Write(label[:upperIndex])
	for i := upperIndex; i < len(label); i++ {
		ch := label[i]
		if ch >= 'A' && ch <= 'Z' {
			dst.WriteByte(ch + ('a' - 'A'))
		} else {
			dst.WriteByte(ch)
		}
	}
}
