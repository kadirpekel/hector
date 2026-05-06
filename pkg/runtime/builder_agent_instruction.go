package runtime

import _ "embed"

const builtInBuilderAgentName = "_hector_builder"

//go:embed assets/builder_instruction.md
var builtInBuilderInstruction string
