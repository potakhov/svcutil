package svcutil

import (
	cryptorand "crypto/rand"
	"encoding/binary"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

const (
	defaultCookieLenK = 32
	letterIdxBits     = 6                    // 6 bits to represent a letter index
	letterIdxMask     = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax      = 63 / letterIdxBits   // # of letter indices fitting in 63 bits

	incrementedSourceOffset = 100000000
)

func CryptoRand(n int) ([]byte, error) {
	x := make([]byte, n)
	_, err := cryptorand.Read(x)
	if err != nil {
		return nil, err
	}
	return x, nil
}

type CookieSource int

const (
	CookieSourcePseudoRand = iota
	CookieSourceCryptoRand
	CookieSourceCustomSnowflake
	CookieSourceIncremented
)

func (cs CookieSource) String() string {
	switch cs {
	case CookieSourcePseudoRand:
		return "CookieSourcePseudoRand"
	case CookieSourceCryptoRand:
		return "CookieSourceCryptoRand"
	case CookieSourceCustomSnowflake:
		return "CookieSourceCustomSnowflake"
	case CookieSourceIncremented:
		return "CookieSourceIncremented"
	default:
		return fmt.Sprintf("unknown CookieSource: %d", cs)
	}
}

type generator interface {
	getNext() int64
}

type CookieGen struct {
	m   sync.Mutex
	gen generator
	src CookieSource
}

// NewCookieGen creates new generator
func NewCookieGen(src CookieSource, nodeID int64) *CookieGen {
	switch src {
	case CookieSourceIncremented:
		return newIncrementedSource(nodeID)
	case CookieSourcePseudoRand:
		return newCookieSourcePseudoRand()
	case CookieSourceCryptoRand:
		return newCookieSourceCryptoRand()
	default:
		// default to cryptorand
		return newCookieSourceCryptoRand()
	}
}

func NewSnowflakeCookieGen(epoch int64, nodeID int64) *CookieGen {
	return newCookieSourceSnowflake(epoch, nodeID)
}

func (cg *CookieGen) String() string {
	return cg.src.String()

}

type incrementedSource struct {
	id uint64
}

func (cg *incrementedSource) getNext() int64 {
	cg.id++
	return int64(cg.id)
}

func newIncrementedSource(nodeID int64) *CookieGen {
	cookieGen := &CookieGen{}
	gen := &incrementedSource{
		id: uint64(incrementedSourceOffset * nodeID),
	}

	cookieGen.gen = gen
	cookieGen.src = CookieSourceIncremented
	return cookieGen
}

type snowGen struct {
	snowGenerator *SnowflakeNode
}

func (cg *snowGen) getNext() int64 {
	return cg.snowGenerator.Generate().Int64()
}

func newCookieSourceSnowflake(epoch int64, nodeID int64) *CookieGen {
	cookieGen := &CookieGen{}
	snowGenerator, err := NewSnowflakeNode(epoch, nodeID)
	if err != nil {
		gen := &pseudoRand{}
		cookieGen.gen = gen
		cookieGen.src = CookieSourcePseudoRand
		return cookieGen
	}

	gen := &snowGen{}
	gen.snowGenerator = snowGenerator
	cookieGen.gen = gen
	cookieGen.src = CookieSourceCustomSnowflake
	return cookieGen

}

type pseudoRand struct {
	pseudoRand rand.Source
}

func (cg *pseudoRand) getNext() int64 {
	return cg.pseudoRand.Int63()
}

func newCookieSourcePseudoRand() *CookieGen {
	cookieGen := &CookieGen{}
	gen := &pseudoRand{}
	gen.pseudoRand = rand.NewSource(time.Now().UnixNano())
	cookieGen.gen = gen
	cookieGen.src = CookieSourcePseudoRand
	return cookieGen
}

type cryptoRand struct {
	fallbackRand rand.Source
}

func (cg *cryptoRand) getNext() int64 {
	b, err := CryptoRand(8)
	if err != nil {
		return cg.fallbackRand.Int63()
	}

	v := binary.BigEndian.Uint64(b)
	return int64(v & ^(uint64(1) << 63))
}

func newCookieSourceCryptoRand() *CookieGen {
	cookieGen := &CookieGen{}
	gen := &cryptoRand{}
	gen.fallbackRand = rand.NewSource(time.Now().UnixNano())
	cookieGen.gen = gen
	cookieGen.src = CookieSourceCryptoRand
	return cookieGen
}

func (cg *CookieGen) getNext() int64 {
	cg.m.Lock()
	defer cg.m.Unlock()
	return cg.gen.getNext()
}

// Cookie produces new string cookie
func (cg *CookieGen) Cookie() string {
	b := make([]byte, defaultCookieLenK)

	for i, cache, remain := defaultCookieLenK-1, cg.getNext(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = cg.getNext(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}

// Int63 produces new int63 cookie packed in uint64
func (cg *CookieGen) Int63() uint64 {
	return uint64(cg.getNext())
}

func (cg *CookieGen) CookieSource() CookieSource {
	return cg.src
}
