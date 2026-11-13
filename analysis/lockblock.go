// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package analysis

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"maps"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/buildssa"
	"golang.org/x/tools/go/ssa"
)

// The blocking operations as a report names them.
const (
	sendOp     = "a send to a channel"
	receiveOp  = "a receive from a channel"
	selectOp   = "a select without a default case"
	sleepOp    = "time.Sleep"
	groupOp    = "sync.WaitGroup.Wait"
	condOp     = "sync.Cond.Wait"
	blockingOp = "%s of a blocking interface"
	hookOp     = "the hook %s"
)

// The messages of a finding of lockblock: a blocking operation, and a call of a function that blocks
// with the path of its calls.
const (
	operationMessage = "%s blocks inside the critical section of %s from line %d"
	callMessage      = "the call of %s blocks inside the critical section of %s from line %d: %s"
)

// pathSeparator separates the calls of a path in a finding of lockblock.
const pathSeparator = " -> "

// LockBlock returns the analyzer lockblock. It reports each blocking operation inside the critical
// section of a sync.Mutex or a sync.RWMutex, and each call inside one of a function that blocks,
// with the path of calls to the blocking operation.
//
// A critical section begins at a call of Lock or RLock, also of an embedded lock, and ends at the
// call of Unlock or RUnlock of the same lock in the same function. A deferred Unlock ends it when
// the function returns. Two calls lock the same lock when their receivers have the same access
// path: the same variable, then the same fields, dereferences and index values. A position is
// inside the critical section when every path to it holds the lock.
//
// A blocking operation is a send, a receive, a select without a default case, a call of
// time.Sleep or sync.WaitGroup.Wait, a call of sync.Cond.Wait while two or more locks are held,
// and a call of a method of an interface that the flag interfaces or the flag hooks names. A
// function blocks when its body makes a blocking operation or calls a function that blocks. The
// analyzer exports that summary as a fact, which the analysis of each importing package reads. A
// call through any other interface, of a function value, in a go statement or in a defer
// statement is not followed. A call into the runtime or into an internal package of the standard
// library is not followed either.
//
// An Unlock or RUnlock unlocks a lock of the caller when the function has not locked that lock on
// any path to the call. The blocking operations between it and the next Lock or RLock of that lock
// do not block inside its critical section. The analyzer assumes that a function unlocks the locks that its caller locked
// last. A call of such a function therefore blocks inside the critical section of a lock that the
// caller locked before them.
//
// Each flag takes a list that commas separate, of an import path, a dot and the name of an
// interface, such as io.Writer or example.com/disk.Device. A call matches through the interface
// that declares its method. The analyzer reports in the files other than test files and
// generated files, and skips every package that skip reports. A function of a generated file
// still has a summary.
func LockBlock(skip Skip) *analysis.Analyzer {
	interfaces, hooks := names{}, names{}
	a := &analysis.Analyzer{
		Name:      "lockblock",
		Doc:       "report each blocking call inside the critical section of a mutex",
		URL:       "https://pkg.go.dev/go.dokimi.dev/lint/analysis#LockBlock",
		Requires:  []*analysis.Analyzer{buildssa.Analyzer},
		FactTypes: []analysis.Fact{new(blocks)},
		Run: func(pass *analysis.Pass) (any, error) {
			pkg := pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA) //nolint:forcetypeassert // Requires names buildssa
			c := &checker{
				pass:       pass,
				pkg:        pkg,
				interfaces: interfaces,
				hooks:      hooks,
				summaries:  map[*ssa.Function]*blocks{},
				states:     map[*ssa.Function][]lockState{},
			}
			c.run(skip)
			return nil, nil //nolint:nilnil // the analyzer reports diagnostics and returns no result
		},
	}
	a.Flags.Var(
		interfaces,
		"interfaces",
		"the `interfaces` whose methods block, such as io.Writer,example.com/disk.Device",
	)
	a.Flags.Var(
		hooks,
		"hooks",
		"the `interfaces` whose methods run code outside the module, such as the hook of a plugin",
	)
	return a
}

// names are the interfaces of a flag of lockblock, each an import path, a dot and the name of the
// interface.
type names map[string]bool

// String returns the names of n, sorted and separated by commas.
func (n names) String() string {
	return strings.Join(slices.Sorted(maps.Keys(n)), ",")
}

// Set adds each name of value, a list that commas separate, to n. It returns an error for an entry
// without an import path or without the name of an interface.
func (n names) Set(value string) error {
	for name := range strings.SplitSeq(value, ",") {
		dot := strings.LastIndex(name, ".")
		if dot <= 0 || dot == len(name)-1 {
			return fmt.Errorf("lockblock: the entry %q is no import path, a dot and an interface name", name)
		}
		n[name] = true
	}
	return nil
}

// blocks is the summary of a function that blocks, and its fact: the functions that its path of
// calls passes, from its callee on, with the blocking operation last, and the number of locks that
// the callers hold at a call for the operation to block inside one of their critical sections.
type blocks struct {
	// Path is the path of calls.
	Path []string
	// Locks is the number of locks that the callers hold at a call for the path to block inside a
	// critical section of one of them. It is 1 or more.
	Locks int
}

// AFact marks blocks as a fact of lockblock.
func (*blocks) AFact() {}

// String returns the path of b as a finding states it, and the number of locks when it is more
// than 1.
func (b *blocks) String() string {
	path := strings.Join(b.Path, pathSeparator)
	if b.Locks > 1 {
		return fmt.Sprintf("%s when its callers hold %d locks", path, b.Locks)
	}
	return path
}

// heldLock is a lock that a critical section holds: the access path of its receiver, and the
// position of the call that locked it.
type heldLock struct {
	// key is the access path of the receiver.
	key lockKey
	// pos is the position of the call of Lock or RLock.
	pos token.Pos
}

// lockKey is the access path of the receiver of a call of a lock: the value at its root, and the
// fields, dereferences and index values after it.
type lockKey struct {
	// root is the variable, the parameter, the free variable or the global of the access path.
	root ssa.Value
	// path is the fields, dereferences and index values after root.
	path string
}

// lockState is the state of the locks of a function at a position: the locks of its own that every
// path to the position holds, the locks that a path to it locks or tries, and the locks of its
// callers that a path to it unlocks.
type lockState struct {
	// held are the locks that every path holds, in the order of their calls of Lock and RLock.
	held []heldLock
	// locked are the locks that a path locks or tries and does not unlock again.
	locked []lockKey
	// released are the locks of the callers that a path unlocks and does not lock again.
	released []lockKey
}

// checker is one run of lockblock over a package.
type checker struct {
	// pass is the pass of the run.
	pass *analysis.Pass
	// pkg is the SSA form of the package.
	pkg *buildssa.SSA
	// interfaces and hooks are the interfaces of the flags.
	interfaces, hooks names
	// summaries are the summaries of the functions of the package that block.
	summaries map[*ssa.Function]*blocks
	// states are the states of the locks at the start of each block, by the index of the block,
	// for each function of the package.
	states map[*ssa.Function][]lockState
	// receivers are the receivers of the calls of Lock and RLock as the source writes them, by the
	// position of the parenthesis of the call.
	receivers map[token.Pos]string
}

// run summarizes the functions of the package, exports a fact for each declared function that
// blocks, and reports the blocking operations and calls inside the critical sections of the files
// that it reports in. A summary changes only to a summary with fewer locks, so the summaries of
// recursive functions settle.
func (c *checker) run(skip Skip) {
	for _, fn := range c.pkg.SrcFuncs {
		c.states[fn] = entries(fn)
	}
	for changed := true; changed; {
		changed = false
		for _, fn := range c.pkg.SrcFuncs {
			if s := c.summary(fn); s != nil && (c.summaries[fn] == nil || s.Locks < c.summaries[fn].Locks) {
				c.summaries[fn], changed = s, true
			}
		}
	}
	c.export()
	reported := map[*token.File]bool{}
	c.receivers = map[token.Pos]string{}
	for _, f := range scope(c.pass, skip) {
		if !isTest(c.pass, f) {
			reported[c.pass.Fset.File(f.Pos())] = true
			c.readReceivers(f)
		}
	}
	for _, fn := range c.pkg.SrcFuncs {
		if reported[c.pass.Fset.File(fn.Pos())] {
			c.report(fn)
		}
	}
}

// export exports the fact of each declared function of the package that blocks, for the packages
// that import it. A function of a test file and a package main have no importer, so they export
// none.
func (c *checker) export() {
	if c.pass.Pkg.Name() == "main" {
		return
	}
	tests := map[*token.File]bool{}
	for _, f := range c.pass.Files {
		tests[c.pass.Fset.File(f.Pos())] = isTest(c.pass, f)
	}
	for _, fn := range c.pkg.SrcFuncs {
		f, ok := fn.Object().(*types.Func)
		if ok && c.summaries[fn] != nil && !tests[c.pass.Fset.File(fn.Pos())] {
			c.pass.ExportObjectFact(f, c.summaries[fn])
		}
	}
}

// readReceivers adds the receiver of each call of Lock and RLock of f to the receivers of c: the
// operand of the selector, and the first argument of a method expression such as
// (*sync.Mutex).Lock(mu).
func (c *checker) readReceivers(f *ast.File) {
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok && (sel.Sel.Name == "Lock" || sel.Sel.Name == "RLock") {
			receiver := sel.X
			if c.pass.TypesInfo.Types[sel.X].IsType() {
				receiver = call.Args[0]
			}
			c.receivers[call.Lparen] = types.ExprString(receiver)
		}
		return true
	})
}

// summary returns the summary of fn, and nil when fn makes no blocking operation. Its path leads to
// the first operation in the order of the instructions that blocks with the fewest locks of the
// callers held. An operation needs one more lock of the callers for each lock of theirs that fn
// unlocks before it, and the locks that fn holds count toward a sync.Cond.Wait.
func (c *checker) summary(fn *ssa.Function) *blocks {
	var best *blocks
	for i, b := range fn.Blocks {
		state := c.states[fn][i]
		for _, instr := range b.Instrs {
			if path, locks := c.blocking(instr); path != nil {
				locks = len(state.released) + max(1, locks-len(state.held))
				if best == nil || locks < best.Locks {
					best = &blocks{Path: path, Locks: locks}
				}
			}
			state = step(state, instr)
		}
	}
	return best
}

// report reports each blocking operation and each call of a function that blocks inside a critical
// section of fn, once for each position. It names the lock that fn locked last of the locks that
// are still held during the operation.
func (c *checker) report(fn *ssa.Function) {
	reported := map[token.Pos]bool{}
	for i, b := range fn.Blocks {
		state := c.states[fn][i]
		for _, instr := range b.Instrs {
			pos := instr.Pos()
			if len(state.held) > 0 && pos.IsValid() && !reported[pos] {
				if path, locks := c.blocking(instr); path != nil && len(state.held) >= locks {
					reported[pos] = true
					c.reportAt(pos, state.held[len(state.held)-locks], path)
				}
			}
			state = step(state, instr)
		}
	}
}

// reportAt reports the path of a blocking operation or call at pos inside the critical section of
// lock.
func (c *checker) reportAt(pos token.Pos, lock heldLock, path []string) {
	receiver := c.receivers[lock.pos]
	line := c.pass.Fset.Position(lock.pos).Line
	if len(path) == 1 {
		c.pass.Reportf(pos, operationMessage, path[0], receiver, line)
		return
	}
	c.pass.Reportf(pos, callMessage, path[0], receiver, line, strings.Join(path, pathSeparator))
}

// blocking returns the path of instr when it blocks, and nil otherwise, with the number of locks
// held at instr for it to block inside a critical section. The path is the operation for a
// blocking operation, and the callee and the path of its calls for a call of a function that
// blocks.
func (c *checker) blocking(instr ssa.Instruction) ([]string, int) {
	switch instr := instr.(type) {
	case *ssa.Send:
		return []string{sendOp}, 1
	case *ssa.UnOp:
		if instr.Op == token.ARROW {
			return []string{receiveOp}, 1
		}
	case *ssa.Select:
		if instr.Blocking {
			return []string{selectOp}, 1
		}
	case *ssa.Call:
		return c.call(instr.Common())
	}
	return nil, 0
}

// call returns the path of a call that blocks, with the number of locks held at the call for it to
// block inside a critical section, and nil for a call that does not block or that the analyzer
// does not follow. A sync.Cond.Wait needs two locks, because it unlocks the lock of its cond while
// it waits.
func (c *checker) call(common *ssa.CallCommon) ([]string, int) {
	if common.IsInvoke() {
		return c.invoke(common.Method), 1
	}
	callee := common.StaticCallee()
	if callee == nil {
		return nil, 0
	}
	f, _ := callee.Object().(*types.Func)
	switch op := stdOperation(f); {
	case op == condOp:
		return []string{op}, 2
	case op != "":
		return []string{op}, 1
	case f != nil && f.Pkg() != nil && unfollowed(f.Pkg()):
		return nil, 0
	}
	if s := c.summaries[callee]; s != nil {
		return append([]string{calleeName(callee)}, s.Path...), s.Locks
	}
	var fact blocks
	if f != nil && callee.Pkg != c.pkg.Pkg && c.pass.ImportObjectFact(f, &fact) {
		return append([]string{funcName(f)}, fact.Path...), fact.Locks
	}
	return nil, 0
}

// invoke returns the path of a call of the method m of an interface, which blocks when the flag
// interfaces or the flag hooks names the interface that declares m.
func (c *checker) invoke(m *types.Func) []string {
	named, ok := m.Signature().Recv().Type().(*types.Named)
	if !ok || named.Obj().Pkg() == nil {
		return nil
	}
	name := named.Obj().Pkg().Path() + "." + named.Obj().Name()
	switch {
	case c.interfaces[name]:
		return []string{fmt.Sprintf(blockingOp, funcName(m))}
	case c.hooks[name]:
		return []string{fmt.Sprintf(hookOp, funcName(m))}
	}
	return nil
}

// stdOperation returns the blocking operation of the standard library that f is, and the empty
// string for any other function: time.Sleep, sync.WaitGroup.Wait and sync.Cond.Wait.
func stdOperation(f *types.Func) string {
	if f == nil || f.Pkg() == nil {
		return ""
	}
	name := f.Name()
	if recv := f.Signature().Recv(); recv != nil {
		unqualified := types.TypeString(recv.Type(), func(*types.Package) string { return "" })
		name = strings.TrimPrefix(unqualified, "*") + "." + name
	}
	switch op := f.Pkg().Path() + "." + name; op {
	case sleepOp, groupOp, condOp:
		return op
	}
	return ""
}

// unfollowed reports whether the analyzer follows no call into pkg: the runtime and the internal
// packages of the standard library. Their channel operations belong to the scheduler and the
// collector, which every allocation may enter.
func unfollowed(pkg *types.Package) bool {
	path := pkg.Path()
	return path == "runtime" || strings.HasPrefix(path, "runtime/") || strings.HasPrefix(path, "internal/")
}

// lockOp is the operation of a call of a method of a sync.Mutex or a sync.RWMutex.
type lockOp int

// The operations of a call: none, Lock or RLock, TryLock or TryRLock, and Unlock or RUnlock.
const (
	opNone lockOp = iota
	opLock
	opTryLock
	opUnlock
)

// lockCall returns the receiver and the operation of a call of a method of a sync.Mutex or a
// sync.RWMutex that locks, tries or unlocks it, and opNone for any other instruction. A call of a
// method value passes no receiver, because the wrapper of SSA keeps it, so it is not such a call.
func lockCall(instr ssa.Instruction) (ssa.Value, lockOp) {
	call, ok := instr.(*ssa.Call)
	if !ok || len(call.Common().Args) == 0 {
		return nil, opNone
	}
	callee := call.Common().StaticCallee()
	if callee == nil {
		return nil, opNone
	}
	f, ok := callee.Object().(*types.Func)
	if !ok || f.Pkg() == nil || f.Pkg().Path() != "sync" || f.Signature().Recv() == nil {
		return nil, opNone
	}
	recv := types.TypeString(f.Signature().Recv().Type(), qualifier)
	if recv != "*sync.Mutex" && recv != "*sync.RWMutex" {
		return nil, opNone
	}
	switch f.Name() {
	case "Lock", "RLock":
		return call.Common().Args[0], opLock
	case "TryLock", "TryRLock":
		return call.Common().Args[0], opTryLock
	case "Unlock", "RUnlock":
		return call.Common().Args[0], opUnlock
	}
	return nil, opNone
}

// step returns the state of the locks after instr, for the state before it.
//
//   - Lock and RLock of a lock of the callers that the state releases lock it again for the
//     callers. Of any other lock, they add the lock to held, last, and to locked.
//   - TryLock and TryRLock add the lock to locked.
//   - Unlock and RUnlock of a lock in locked remove it from held and from locked. Of any other
//     lock, they add it to released.
func step(state lockState, instr ssa.Instruction) lockState {
	recv, op := lockCall(instr)
	if op == opNone {
		return state
	}
	key := keyOf(recv)
	switch {
	case op == opLock && slices.Contains(state.released, key):
		state.released = without(state.released, key)
	case op == opLock:
		state.held = append(withoutHeld(state.held, key), heldLock{key: key, pos: instr.Pos()})
		state.locked = with(state.locked, key)
	case op == opTryLock:
		state.locked = with(state.locked, key)
	case slices.Contains(state.locked, key):
		state.held = withoutHeld(state.held, key)
		state.locked = without(state.locked, key)
	default:
		state.released = with(state.released, key)
	}
	return state
}

// keyOf returns the access path of v, the receiver of a call of a lock.
func keyOf(v ssa.Value) lockKey {
	var path []string
	for {
		switch x := v.(type) {
		case *ssa.FieldAddr:
			path, v = append(path, "."+strconv.Itoa(x.Field)), x.X
			continue
		case *ssa.IndexAddr:
			path, v = append(path, "["+x.Index.Name()+"]"), x.X
			continue
		case *ssa.UnOp:
			if x.Op == token.MUL {
				path, v = append(path, "*"), x.X
				continue
			}
		}
		slices.Reverse(path)
		return lockKey{root: v, path: strings.Join(path, "")}
	}
}

// entries returns the state of the locks at the start of each block of fn, by the index of the
// block, over every path to the block. The state of the entry block has no lock, and neither has
// the state of a block that no path reaches.
func entries(fn *ssa.Function) []lockState {
	in := make([]lockState, len(fn.Blocks))
	known := make([]bool, len(fn.Blocks))
	if len(fn.Blocks) == 0 {
		return in
	}
	known[0] = true
	for changed := true; changed; {
		changed = false
		for _, b := range fn.Blocks {
			if !known[b.Index] {
				continue
			}
			out := in[b.Index]
			for _, instr := range b.Instrs {
				out = step(out, instr)
			}
			for _, s := range b.Succs {
				if !known[s.Index] {
					in[s.Index], known[s.Index], changed = out, true, true
					continue
				}
				if merged, differs := join(in[s.Index], out); differs {
					in[s.Index], changed = merged, true
				}
			}
		}
	}
	return in
}

// join returns the state of the locks where a path of the state a meets a path of the state b, and
// whether it differs from a: the locks that both hold, and the locks that either locks or releases.
func join(a, b lockState) (lockState, bool) {
	held := slices.DeleteFunc(slices.Clone(a.held), func(h heldLock) bool {
		return !slices.ContainsFunc(b.held, func(other heldLock) bool { return other.key == h.key })
	})
	locked, released := a.locked, a.released
	for _, key := range b.locked {
		locked = with(locked, key)
	}
	for _, key := range b.released {
		released = with(released, key)
	}
	differs := len(held) != len(a.held) || len(locked) != len(a.locked) || len(released) != len(a.released)
	return lockState{held: held, locked: locked, released: released}, differs
}

// with returns keys with key added last, in a new slice when keys lacks it.
func with(keys []lockKey, key lockKey) []lockKey {
	if slices.Contains(keys, key) {
		return keys
	}
	return append(slices.Clip(keys), key)
}

// without returns keys without key, in a new slice.
func without(keys []lockKey, key lockKey) []lockKey {
	return slices.DeleteFunc(slices.Clone(keys), func(other lockKey) bool { return other == key })
}

// withoutHeld returns held without the lock of key, in a new slice.
func withoutHeld(held []heldLock, key lockKey) []heldLock {
	return slices.DeleteFunc(slices.Clone(held), func(h heldLock) bool { return h.key == key })
}

// qualifier names a package by its name, as a finding names a function.
func qualifier(p *types.Package) string {
	return p.Name()
}

// funcName returns the name of f in a finding: pkg.F, pkg.T.M or (*pkg.T).M.
func funcName(f *types.Func) string {
	recv := f.Signature().Recv()
	if recv == nil {
		return f.Pkg().Name() + "." + f.Name()
	}
	t := types.TypeString(recv.Type(), qualifier)
	if strings.HasPrefix(t, "*") {
		t = "(" + t + ")"
	}
	return t + "." + f.Name()
}

// calleeName returns the name of fn in a finding, and the name of a function literal by the
// function that it is written in.
func calleeName(fn *ssa.Function) string {
	if f, ok := fn.Object().(*types.Func); ok {
		return funcName(f)
	}
	return "a function literal of " + calleeName(fn.Parent())
}
