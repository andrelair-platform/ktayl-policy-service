package documents

import "github.com/andrelair-platform/ktayl-policy-service/internal/domain"

// GeneratorFunc adapts a plain function to the domain.DocumentGenerator interface.
type GeneratorFunc func(p *domain.Policy) ([]byte, error)

func (f GeneratorFunc) GenerateAttestation(p *domain.Policy) ([]byte, error) { return f(p) }
