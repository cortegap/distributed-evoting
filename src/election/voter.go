package election

import (
	"crypto/rand"
	"math/big"
	"sync"

	"6.824/labrpc"
)

type Voter struct {
	mu               sync.Mutex
	committeeMembers []*labrpc.ClientEnd
	voterId          int64
	vote             int
	nShares          int
	shares           []big.Int
	done             bool
	field            int
}

func nrand() int64 {
	max := big.NewInt(int64(1) << 62)
	bigx, _ := rand.Int(rand.Reader, max)
	x := bigx.Int64()
	return x
}

//
// Function to create Shamir Shares
// - Polynomial of degree vt.nShares over field vt.field
// - len(vt.committeeMembers) shares
//
func (vt *Voter) makeShares() {
	vt.mu.Lock()
	defer vt.mu.Unlock()

	// Polynomial of degree nShares with coefficients in field Z_field
	coefficients := make([]big.Int, vt.nShares+1)

	coefficients[0] = *big.NewInt(int64(vt.vote))
	for i := 1; i < vt.nShares; i++ {
		val, _ := rand.Int(rand.Reader, big.NewInt(int64(vt.field)))
		coefficients[i] = *val
	}

	// Compute shares. For each committe member i, evaluate polynomial
	// at x = i + 1
	for i := range vt.shares {
		vt.shares[i] = vt.evalPolynimialL(coefficients, *big.NewInt(int64(i + 1)))
	}

	go vt.SendShares()
}

//
// Evaluate polynomial
// f(x) = coef[0] + coef[0]x + coef[0]x^2 + ...
//
func (vt *Voter) evalPolynimialL(coef []big.Int, x big.Int) big.Int {
	fx := big.NewInt(0)

	for exp, c := range coef {
		var monomial *big.Int
		monomial.Exp(&x, big.NewInt(int64(exp)), nil) // x^exp
		monomial.Mul(monomial, &c)
		fx.Add(fx, monomial)
	}

	return *fx.Mod(fx, big.NewInt(int64(vt.field)))
}

//
// Send votes to the vote counter until the election time closes
//
func (vt *Voter) SendShares() {
	for !vt.Done() {
		return
	}
}

func (vt *Voter) Done() bool {
	vt.mu.Lock()
	defer vt.mu.Unlock()

	return vt.done
}

//
// main/voter.go calls this function.
//
func MakeVoter(committeeMembers []*labrpc.ClientEnd, vote int, nShares int, field int) *Voter {
	vt := &Voter{}

	vt.committeeMembers = committeeMembers
	vt.nShares = nShares
	vt.voterId = nrand()
	vt.vote = vote
	vt.shares = make([]big.Int, len(committeeMembers))
	vt.field = field

	go vt.makeShares()

	return vt
}
