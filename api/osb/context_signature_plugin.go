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

const (
	ContextSignaturePluginName = "ContextSignaturePlugin"
	ServiceInstanceIDFieldName = "service_instance_id"
)

type ContextSignaturePlugin struct {
	contextSigner *ContextSigner
}

type ContextSigner struct {
	ContextPrivateKey string
	rsaPrivateKey     *rsa.PrivateKey
}

func NewCtxSignaturePlugin(contextSigner *ContextSigner) *ContextSignaturePlugin {
	return &ContextSignaturePlugin{
		contextSigner: contextSigner,
	}
}

func (s *ContextSignaturePlugin) Name() string {
	return ContextSignaturePluginName
}

func (s *ContextSignaturePlugin) Provision(req *web.Request, next web.Handler) (*web.Response, error) {
	return s.signContext(req, next)
}

func (s *ContextSignaturePlugin) Bind(req *web.Request, next web.Handler) (*web.Response, error) {
	return s.signContext(req, next)
}

func (s *ContextSignaturePlugin) UpdateService(req *web.Request, next web.Handler) (*web.Response, error) {
	return s.signContext(req, next)
}

func (s *ContextSignaturePlugin) signContext(req *web.Request, next web.Handler) (*web.Response, error) {
	//in case the private key is not provided we continue without adding the signature. this is useful in case we want to toggle off the feature
	if s.contextSigner.ContextPrivateKey == "" {
		log.C(req.Context()).Debugf("context private key not found. context signature can not be calculated")
		return next.Handle(req)
	}
	//unmarshal and marshal the request body so the fields within the context will be ordered lexicographically, and to get rid of redundant spaces\drop-line\tabs
	var reqBodyMap map[string]interface{}
	err := json.Unmarshal(req.Body, &reqBodyMap)
	if err != nil {
		log.C(req.Context()).Errorf("failed to unmarshal context: %v", err)
		return next.Handle(req)
	}
	if _, found := reqBodyMap["context"]; !found {
		errorMsg := "context not found on request body"
		log.C(req.Context()).Error(errorMsg)
		return next.Handle(req)
	}
	contextMap := reqBodyMap["context"].(map[string]interface{})

	if instanceID, ok := req.PathParams[InstanceIDPathParam]; ok {
		contextMap[ServiceInstanceIDFieldName] = instanceID
	}

	err = s.contextSigner.Sign(req.Context(), contextMap)
	if err != nil {
		log.C(req.Context()).Errorf("failed to sign request context: %v", err)
		return next.Handle(req)
	}

	reqBody, err := json.Marshal(reqBodyMap)
	if err != nil {
		log.C(req.Context()).Errorf("failed to marshal request body: %v", err)
		return next.Handle(req)
	}
	req.Body = reqBody

	return next.Handle(req)
}

func (cs *ContextSigner) Sign(ctx context.Context, contextMap map[string]interface{}) error {
	if cs.ContextPrivateKey == "" {
		errorMsg := "context rsa private key is missing. context signature can not be calculated"
		log.C(ctx).Errorf(errorMsg)
		return fmt.Errorf(errorMsg)
	}
	ctxByte, err := json.Marshal(contextMap)
	if err != nil {
		log.C(ctx).Errorf("failed to marshal context: %v", err)
		return err
	}

	//on the first time the sign function is executed we should parse the rsa private key and keep it for next executions
	if cs.rsaPrivateKey == nil {
		cs.rsaPrivateKey, err = cs.parseRsaPrivateKey(ctx, cs.ContextPrivateKey)
		if err != nil {
			log.C(ctx).Errorf("failed to parse rsa private key: %v", err)
			return err
		}
	}
	signedCtx, err := cs.calculateSignature(ctx, string(ctxByte), cs.rsaPrivateKey)
	if err != nil {
		log.C(ctx).Errorf("failed to calculate the context signature: %v", err)
		return err
	}

	contextMap["signature"] = signedCtx
	return nil
}

func (cs *ContextSigner) parseRsaPrivateKey(ctx context.Context, rsaPrivateKey string) (*rsa.PrivateKey, error) {
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

func (cs *ContextSigner) calculateSignature(ctx context.Context, ctxStr string, rsaPrivateKey *rsa.PrivateKey) (string, error) {
	log.C(ctx).Debugf("creating signature for ctx: %s", ctxStr)

	hashedCtx := sha256.Sum256([]byte(ctxStr))

	signature, err := rsa.SignPKCS1v15(rand.Reader, rsaPrivateKey, crypto.SHA256, hashedCtx[:])
	if err != nil {
		log.C(ctx).Errorf("failed to encrypt context %v", err)
		return "", err
	}
	return base64.StdEncoding.EncodeToString(signature), nil
}
