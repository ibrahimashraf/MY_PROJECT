package symbolic

// TokenType classifies lexical elements in a formal engineering specification.
type TokenType int

const (
	TokenIdent  TokenType = iota // Identifier / symbol
	TokenNumber                  // Numeric literal
	TokenOp                      // Operator (+, -, *, /, =, >, <)
	TokenLParen                  // (
	TokenRParen                  // )
	TokenEOF
)

// Token is a tagged lexical unit.
type Token struct {
	Type    TokenType
	Literal string
}

// Tokenize performs basic lexical analysis on a formal specification string.
func Tokenize(input string) []Token {
	var tokens []Token
	i := 0
	for i < len(input) {
		ch := input[i]
		switch {
		case ch == ' ' || ch == '\t' || ch == '\n':
			i++
		case ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch == '_':
			j := i
			for j < len(input) && (input[j] >= 'a' && input[j] <= 'z' ||
				input[j] >= 'A' && input[j] <= 'Z' ||
				input[j] >= '0' && input[j] <= '9' || input[j] == '_') {
				j++
			}
			tokens = append(tokens, Token{Type: TokenIdent, Literal: input[i:j]})
			i = j
		case ch >= '0' && ch <= '9' || ch == '.':
			j := i
			for j < len(input) && (input[j] >= '0' && input[j] <= '9' || input[j] == '.') {
				j++
			}
			tokens = append(tokens, Token{Type: TokenNumber, Literal: input[i:j]})
			i = j
		case ch == '(':
			tokens = append(tokens, Token{Type: TokenLParen, Literal: "("})
			i++
		case ch == ')':
			tokens = append(tokens, Token{Type: TokenRParen, Literal: ")"})
			i++
		default:
			tokens = append(tokens, Token{Type: TokenOp, Literal: string(ch)})
			i++
		}
	}
	tokens = append(tokens, Token{Type: TokenEOF, Literal: ""})
	return tokens
}

// ProofWitness represents a formal verification certificate for a claim.
type ProofWitness struct {
	Claim    string
	Proof    string
	IsValid  bool
	Residual float64 // Numeric proof residual (0 = exact)
}

// VerifyClaim checks an engineering specification claim against a numeric witness bound.
func VerifyClaim(claim string, witness string, numericResidual float64, tolerance float64) ProofWitness {
	return ProofWitness{
		Claim:    claim,
		Proof:    witness,
		IsValid:  numericResidual <= tolerance,
		Residual: numericResidual,
	}
}

// CurryHowardProof represents a trivially valid (tautological) proof term for a proposition.
type CurryHowardProof struct {
	Proposition string
	TermType    string // e.g. "A -> A" (identity type)
	Valid       bool
}

// IdentityProof constructs the trivial Curry-Howard identity proof: id : A -> A.
func IdentityProof(propositionType string) CurryHowardProof {
	return CurryHowardProof{
		Proposition: propositionType,
		TermType:    propositionType + " → " + propositionType,
		Valid:       true,
	}
}
