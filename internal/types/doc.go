package types

// Package types defines the shared data structures that flow between
// the stages of the leanBuildDocker pipeline.
//
// These structures are deliberately separated from any specific stage
// because they are contracts — both the producer and consumer of each
// type live in different packages. Keeping types here prevents
// circular imports and makes the data flow explicit:
//
//	Analyzer produces  → ProjectInfo → consumed by Planner
//	Planner  produces  → BuildPlan   → consumed by Renderer
//
// Types in this package are pure data (no methods, no behavior).
// Logic that operates on them lives in the producer and consumer packages.