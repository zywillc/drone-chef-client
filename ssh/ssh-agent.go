package ssh

import (
	"bytes"
	"net"
	"strings"
	"io/ioutil"
	"path/filepath"
	"log"

	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh"
)
// A tiny wrapper around an agent.Agent to expose the ability to close its
// associated connection on request.
type sshAgent struct {
	agent agent.Agent
	conn  net.Conn
	id    string
}

func (a *sshAgent) Close() error {
	if a.conn == nil {
		return nil
	}

	return a.conn.Close()
}

// Try to read an id file using the id as the file path. Also read the .pub
// file if it exists, as the id file may be encrypted. Return only the file
// data read. We don't need to know what data came from which path, as we will
// try parsing each as a private key, a public key and an authorized key
// regardless.
func idKeyData(id string) [][]byte {
	idPath, err := filepath.Abs(id)
	if err != nil {
		return nil
	}

	var fileData [][]byte

	paths := []string{idPath}

	if !strings.HasSuffix(idPath, ".pub") {
		paths = append(paths, idPath+".pub")
	}

	for _, p := range paths {
		d, err := ioutil.ReadFile(p)
		if err != nil {
			log.Printf("[DEBUG] error reading %q: %s", p, err)
			continue
		}
		log.Printf("[DEBUG] found identity data at %q", p)
		fileData = append(fileData, d)
	}

	return fileData
}
// make an attempt to either read the identity file or find a corresponding
// public key file using the typical openssh naming convention.
// This returns the public key in wire format, or nil when a key is not found.
func findIDPublicKey(id string) []byte {
	for _, d := range idKeyData(id) {
		signer, err := ssh.ParsePrivateKey(d)
		if err == nil {
			log.Println("[DEBUG] parsed id private key")
			pk := signer.PublicKey()
			return pk.Marshal()
		}

		// try it as a publicKey
		pk, err := ssh.ParsePublicKey(d)
		if err == nil {
			log.Println("[DEBUG] parsed id public key")
			return pk.Marshal()
		}

		// finally try it as an authorized key
		pk, _, _, _, err = ssh.ParseAuthorizedKey(d)
		if err == nil {
			log.Println("[DEBUG] parsed id authorized key")
			return pk.Marshal()
		}
	}

	return nil
}


// sortSigners moves a signer with an agent comment field matching the
// agent_identity to the head of the list when attempting authentication. This
// helps when there are more keys loaded in an agent than the host will allow
// attempts.
func (s *sshAgent) sortSigners(signers []ssh.Signer) {
	if s.id == "" || len(signers) < 2 {
		return
	}

	// if we can locate the public key, either by extracting it from the id or
	// locating the .pub file, then we can more easily determine an exact match
	idPk := findIDPublicKey(s.id)

	// if we have a signer with a connect field that matches the id, send that
	// first, otherwise put close matches at the front of the list.
	head := 0
	for i := range signers {
		pk := signers[i].PublicKey()
		k, ok := pk.(*agent.Key)
		if !ok {
			continue
		}

		// check for an exact match first
		if bytes.Equal(pk.Marshal(), idPk) || s.id == k.Comment {
			signers[0], signers[i] = signers[i], signers[0]
			break
		}

		// no exact match yet, move it to the front if it's close. The agent
		// may have loaded as a full filepath, while the config refers to it by
		// filename only.
		if strings.HasSuffix(k.Comment, s.id) {
			signers[head], signers[i] = signers[i], signers[head]
			head++
			continue
		}
	}

	ss := []string{}
	for _, signer := range signers {
		pk := signer.PublicKey()
		k := pk.(*agent.Key)
		ss = append(ss, k.Comment)
	}
}

func (s *sshAgent) Signers() ([]ssh.Signer, error) {
	signers, err := s.agent.Signers()
	if err != nil {
		return nil, err
	}

	s.sortSigners(signers)
	return signers, nil
}

func (a *sshAgent) Auth() ssh.AuthMethod {
	return ssh.PublicKeysCallback(a.Signers)
}

func (a *sshAgent) ForwardToAgent(client *ssh.Client) error {
	return agent.ForwardToAgent(client, a.agent)
}
