// Package mocks contains handwritten mock implementations of domain ports.
//
// Mock convention:
//   - File name: <portname>_mock.go (e.g., session_repository_mock.go)
//   - Type name: Mock<PortName> (e.g., MockSessionRepository)
//   - Each method records call count + last args in struct fields
//   - Each method can return canned error via <Method>Err field
//   - NO test framework dependency (these mocks are usable from any test)
package mocks
