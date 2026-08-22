package analyse

import "testing"

func TestObjectTypeAcceptsPhpunitDoubleWithUnboundTemplate(t *testing.T) {
	for _, marker := range []string{
		`PHPUnit\Framework\MockObject\MockObject`,
		`PHPUnit\Framework\MockObject\Stub`,
		`PHPUnit\Framework\MockObject\StubInternal`,
	} {
		t.Run(marker, func(t *testing.T) {
			declared := ParseType(`Example\Domain\Record`)
			actual := ParseType(marker + `&PHPUnit\Framework\RealInstanceType`)

			if !declared.Accepts(actual) {
				t.Fatalf("expected generated test-double intersection to satisfy an object type")
			}
		})
	}
}

func TestScalarTypeRejectsPhpunitMockWithUnboundTemplate(t *testing.T) {
	declared := ParseType("string")
	actual := ParseType(`PHPUnit\Framework\MockObject\MockObject&PHPUnit\Framework\RealInstanceType`)

	if declared.Accepts(actual) {
		t.Fatalf("expected generated mock intersection not to satisfy a scalar type")
	}
}
