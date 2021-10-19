package osb

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"

	//"encoding/json"
	"encoding/pem"
	"fmt"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/web"
)

const ContextSignaturePluginName = "ContextSignaturePlugin"

type ContextSignaturePlugin struct {
	ContextPrivateKey string
	ContextPublicKey  string
}

func NewCtxSignaturePlugin(publicKey, privateKey string) *ContextSignaturePlugin {
	return &ContextSignaturePlugin{
		ContextPrivateKey: privateKey,
		ContextPublicKey:  publicKey,
	}
}

func (s *ContextSignaturePlugin) Name() string {
	return ContextSignaturePluginName
}

func (s *ContextSignaturePlugin) Provision(req *web.Request, next web.Handler) (*web.Response, error) {
	return s.sign(req, next)
}

func (s *ContextSignaturePlugin) Bind(req *web.Request, next web.Handler) (*web.Response, error) {
	return s.sign(req, next)
}

func (s *ContextSignaturePlugin) UpdateService(req *web.Request, next web.Handler) (*web.Response, error) {
	return s.sign(req, next)
}

func (s *ContextSignaturePlugin) sign(req *web.Request, next web.Handler) (*web.Response, error) {
	if s.ContextPrivateKey == "" || s.ContextPublicKey == "" {
		log.C(req.Context()).Errorf("ctx private key or ctx public key is missing. signature will not be added to context")
		return next.Handle(req)
	}

	//unmarshal and marshal the request body so the fields within the context will be ordered lexicographically, and to get rid of redundant spaces\drop-line\tabs
	var reqBodyMap map[string]interface{}
	if err := json.Unmarshal(req.Body, &reqBodyMap); err != nil {
		log.C(req.Context()).Errorf("failed to unmarshal context: %v", err)
		return next.Handle(req)
	}
	if _, found := reqBodyMap["context"]; !found {
		log.C(req.Context()).Error("context not found on request body")
		return next.Handle(req)
	}
	ctxByte, err := json.Marshal(reqBodyMap["context"])
	if err != nil {
		log.C(req.Context()).Errorf("failed to marshal context: %v", err)
		return next.Handle(req)
	}

	signedCtx, err := CalculateSignature(req.Context(), string(ctxByte), s.ContextPrivateKey)
	if err != nil {
		return next.Handle(req)
	}
	ctx := reqBodyMap["context"].(map[string]interface{})
	ctx["signature"] = signedCtx
	reqBodyMap["context"] = ctx

	reqBody, err := json.Marshal(reqBodyMap)
	if err != nil {
		log.C(req.Context()).Errorf("failed to marshal request body %v", err)
		return next.Handle(req)
	}
	req.Body = reqBody

	return next.Handle(req)
}

func CalculateSignature(ctx context.Context, ctxStr, privateKeyStr string) (string, error) {
	log.C(ctx).Debugf("creating signature for ctx: %s", ctxStr)
	privateKey, err := parseRsaPrivateKey(ctx, privateKeyStr)
	if err != nil {
		return "", err
	}

	hashedCtx := sha256.Sum256([]byte(ctxStr))

	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hashedCtx[:])
	if err != nil {
		log.C(ctx).Errorf("failed to encrypt context %v", err)
		return "", err
	}
	return base64.StdEncoding.EncodeToString(signature), nil
}

func parseRsaPrivateKey(ctx context.Context, rsaPrivateKey string) (*rsa.PrivateKey, error) {
	key, err := base64.StdEncoding.DecodeString(rsaPrivateKey)
	if err != nil {
		log.C(ctx).Errorf("failed to base64 decode rsa private key: %v", err)
		return nil, err
	}
	block, _ := pem.Decode(key)
	if block == nil {
		log.C(ctx).Error("failed to pem decode rsa private key")
		return nil, fmt.Errorf("failed to pem decode context rsa private key")
	}
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		log.C(ctx).Errorf("fail to parse rsa key, %s", err.Error())
		return nil, err
	}

	return privateKey, nil
}
