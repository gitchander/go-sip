package server

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/gitchander/go-sip/sipnet"
)

// TODO: Place this in a configuration file
var hostname = "localhost"

type authSession struct {
	nonce   string
	user    sipnet.User
	conn    *sipnet.Conn
	created time.Time
}

// a map[call id]authSession pair
var authSessions = make(map[string]authSession)
var authSessionMutex = new(sync.Mutex)

// ErrInvalidAuthHeader is returned when the Authorization header fails to be
// parsed.
var ErrInvalidAuthHeader = errors.New("server: invalid authorization header")

func generateNonce(size int) string {
	bytes := make([]byte, size)
	_, err := rand.Read(bytes)
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes)
}

func parseAuthHeader(header string) (sipnet.HeaderArgs, error) {
	if len(header) < 8 || strings.ToLower(header[:7]) != "digest " {
		return nil, ErrInvalidAuthHeader
	}

	return sipnet.ParsePairs(header[7:]), nil
}

func requestAuthentication(request *sipnet.Request, conn *sipnet.Conn, from sipnet.User) {
	resp := sipnet.NewResponse(request)

	callID := request.Header.Get("Call-ID")
	if callID == "" {
		resp.BadRequest(conn, "Missing required Call-ID header.")
		return
	}

	nonce := generateNonce(32)

	resp.StatusCode = sipnet.StatusUnauthorized
	// No auth header, deny.
	resp.Header.Set("From", from.String())
	from.Arguments.Del("tag")
	resp.Header.Set("To", from.String())

	authArgs := make(sipnet.HeaderArgs)
	authArgs.Set("realm", hostname)
	authArgs.Set("qop", "auth")
	authArgs.Set("nonce", nonce)
	authArgs.Set("opaque", "")
	authArgs.Set("stale", "FALSE")
	authArgs.Set("algorithm", "MD5")
	resp.Header.Set("WWW-Authenticate", "Digest "+authArgs.CommaString())

	authSessionMutex.Lock()
	authSessions[callID] = authSession{
		nonce:   nonce,
		user:    from,
		conn:    conn,
		created: time.Now(),
	}
	authSessionMutex.Unlock()

	resp.WriteTo(conn)
	return
}

func md5Hex(data string) string {
	sum := md5.Sum([]byte(data))
	return hex.EncodeToString(sum[:])
}

func checkAuthorization(request *sipnet.Request, conn *sipnet.Conn,
	authArgs sipnet.HeaderArgs, user sipnet.User) {
	callID := request.Header.Get("Call-ID")
	authSessionMutex.Lock()
	session, found := authSessions[callID]
	authSessionMutex.Unlock()
	if !found {
		requestAuthentication(request, conn, user)
		return
	}

	if authArgs.Get("username") != user.URI.Username {
		requestAuthentication(request, conn, user)
		return
	}

	if authArgs.Get("nonce") != session.nonce {
		requestAuthentication(request, conn, user)
		return
	}

	username := user.URI.Username
	account, found := accounts[username]
	if !found {
		requestAuthentication(request, conn, user)
		return
	}

	ha1 := md5Hex(username + ":" + hostname + ":" + account.password)
	ha2 := md5Hex(sipnet.MethodRegister + ":" + authArgs.Get("uri"))
	response := md5Hex(ha1 + ":" + session.nonce + ":" + authArgs.Get("nc") +
		":" + authArgs.Get("cnonce") + ":auth:" + ha2)

	if response != authArgs.Get("response") {
		requestAuthentication(request, conn, user)
		return
	}

	if request.Header.Get("Expires") == "0" {
		registeredUsersMutex.Lock()
		delete(registeredUsers, username)
		registeredUsersMutex.Unlock()
		println("logged out " + username)
	} else {
		registerUser(session)
		println("registered " + username)
	}

	resp := sipnet.NewResponse(request)
	resp.StatusCode = sipnet.StatusOK
	resp.Header.Set("From", user.String())

	user.Arguments.Set("tag", generateNonce(5))
	resp.Header.Set("To", user.String())
	resp.WriteTo(conn)

	return
}

// HandleRegister handles REGISTER SIP requests.
func HandleRegister(request *sipnet.Request, conn *sipnet.Conn) {
	from, to, err := sipnet.ParseUserHeader(request.Header)
	if err != nil {
		resp := sipnet.NewResponse(request)
		resp.BadRequest(conn, "Failed to parse From or To header.")
		return
	}

	if to.URI.UserDomain() != from.URI.UserDomain() {
		resp := sipnet.NewResponse(request)
		resp.BadRequest(conn, "User in To and From fields do not match.")
		return
	}

	authHeader := request.Header.Get("Authorization")
	if authHeader == "" {
		requestAuthentication(request, conn, from)
		return
	}

	args, err := parseAuthHeader(authHeader)
	if err != nil {
		resp := sipnet.NewResponse(request)
		resp.BadRequest(conn, "Failed to parse Authorization header.")
		return
	}

	checkAuthorization(request, conn, args, from)
}

func registrationJanitor() {
	for {
		authSessionMutex.Lock()
		for callID, session := range authSessions {
			if time.Now().Sub(session.created) > time.Second*30 {
				delete(authSessions, callID)
			}
		}
		authSessionMutex.Unlock()
		time.Sleep(time.Second * 10)
	}
}

func init() {
	go registrationJanitor()
}
