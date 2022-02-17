// Copyright 2020 ZUP IT SERVICOS EM TECNOLOGIA E INOVACAO SA
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ir

import (
	"fmt"

	"github.com/ZupIT/horusec-engine/internal/ast"
)

// Member is a member of a file, like functions, global variables and constants.
type Member interface {
	member()

	// Name returns the declared name of the package member
	Name() string
}

// Value is an IR value that can be referenced by an instruction.
type Value interface {
	value()

	// Name returns the name of this value, and determines how
	// this Value appears when used as an operand of an
	// Instruction.
	//
	// This is the same as the source name for Parameters,
	// Functions and Vars.
	// For constants, it is a representation of the constant's value
	Name() string
}

// Instruction is an IR instruction that computes a new Value or has some effect.
//
// An Instruction that defines a value also implements the Value interface, an
// Instruction that only has an effect does not.
type Instruction interface {
	instr()

	// String returns the disassembled form of this value.
	String() string
}

// File represents a single file converted to IR representation.
type File struct {
	name    string            // Name of file.
	Members map[string]Member // All file members keyed by name.

	expresions []ast.Expr                 // Top level file expressions.
	imported   map[string]*ExternalMember // All importable packages, keyed by import path.
}

// ExternalMember represents a member that is declared outside the file that is being used.
//
// ExternalMember is created from imports of a single file.
type ExternalMember struct {
	name  string // Named import member.
	Path  string // Full import path of member.
	Alias string // Alias declared on import; or empty.
}

// BasicBlock represents an IR basic block.
type BasicBlock struct {
	Index   int           // Index of this block on Function that this block belongs.
	Comment string        // Optional label; no semantic significance
	Instrs  []Instruction // Instructions in order.
}

// Function represents a function or method with the parameters and signature.
type Function struct {
	name      string          // Function name.
	File      *File           // File that this function belongs.
	Signature *Signature      // Function signature.
	Locals    map[string]*Var // Local variables of this function.
	Blocks    []*BasicBlock   // Basic blocks of the function; nil => external function.

	syntax ast.Node // AST node that represents the Function.

	// The following fields are set transiently during building,
	// then cleared.
	currentBlock *BasicBlock // Where to add instructions.
}

// Signature represents a function or method signature.
type Signature struct {
	Params  []*Parameter // Parameters from left to right; or nil
	Results []*Parameter // Return values from left to right; or nil
}

// Parameter represents an input parameter of a function or method.
type Parameter struct {
	node
	name  string // Name of parameter.
	Value Value  // Default value of parameter or nil.

	parent *Function // Function that the this parameter belongs.
}

// Const represents the value of a constant expression.
type Const struct {
	node
	Value string // Value of constant.
}

// Var represents a variable declaration.
type Var struct {
	node
	name  string // Name of variable.
	Value Value  // Value of variable
}

// Call instruction represents a function or method call.
type Call struct {
	node
	Parent   *Function // Function that Call is inside.
	Function *Function // The function that is being called.
	Args     []Value   // The call function parameters.
}

// node is a mix-in embedded by all IR nodes to provide source code information.
//
// Since node is embedded by all IR nodes (Value's and Instruction's), these nodes
// implements the ast.Node interface which provide the original source code location.
type node struct {
	syntax ast.Node // Original code used to create the IR Value/Instruction.
}

func (n node) Pos() ast.Position { return n.syntax.Pos() }

func (c *Const) value()       {}
func (c *Const) Name() string { return fmt.Sprintf("%q", c.Value) }

func (*Var) value()           {}
func (*Var) instr()           {}
func (v *Var) Name() string   { return v.name }
func (v *Var) String() string { return v.Value.Name() }

func (*Parameter) value()         {}
func (p *Parameter) Name() string { return p.name }

func (*Call) instr() {}

func (*Function) member()        {}
func (m *Function) Name() string { return m.name }

func (*File) member()        {}
func (f *File) Name() string { return f.name }

func (*ExternalMember) member() {}
func (m *ExternalMember) Name() string {
	if m.Alias != "" {
		return m.Alias
	}

	return m.name
}

// Func returns the file-level function of the specified name or nil
// if not found.
//
// If the specified name is a imported function, Func will instantiate
// a new Function object using the path and name of imported function
// as a function name.
//
// nolint:funlen // There is no need to break this method.
func (f *File) Func(name string) *Function {
	fn, exists := f.Members[name]
	if !exists {
		return nil
	}

	switch fn := fn.(type) {
	case *Function:
		return fn
	case *ExternalMember:
		// TODO(matheus): Deal with empty ExternalMember.name fields.
		return &Function{
			name: fmt.Sprintf("%s.%s", fn.Path, fn.name),
			File: f,
		}
	default:
		panic(fmt.Sprintf("ir.File.Func: unexpected function member %T", fn))
	}
}

// ImportedPackage returns the importable package to a given name.
func (f *File) ImportedPackage(name string) *ExternalMember {
	return f.imported[name]
}

// lookup return the declared variable with the given name.
func (fn *Function) lookup(name string) *Var {
	return fn.Locals[name]
}
