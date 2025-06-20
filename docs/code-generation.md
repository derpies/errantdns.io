# Code Generation Guidelines

## Chunk Size and Scope

### File Organization

- Generate complete, functional files that handle all concerns for a single responsibility
- Bundle related code into single files when they share the same conceptual domain (e.g., all config logic in one file)
- When unsure about file organization or if multiple files might be needed, ask before proceeding
- Aim for "conceptually complete" implementations rather than artificially limiting by line count

### Implementation Completeness

- "Functional" means "conceptually complete" - the implementation should address all intended concerns for that component
- It's acceptable for files to depend on each other and require cross-file updates
- Avoid creating implementations that require frequent multi-file rewrites to address single issues
- Balance completeness with maintainability during the creation process

## Review and Iteration Process

### Checkpoints

- Establish review checkpoints after significant changes to any single file or group of related files
- Always checkpoint before major architectural changes or large refactoring efforts
- Generate one complete file or logical unit at a time for review before moving to the next component

### Dependency Handling

- Build components in logical dependency order when possible
- If a file requires imports from not-yet-created internal packages, either:
  - Create the dependencies first, or
  - Ask about the preferred approach for handling the dependency chain

### Code Quality Standards

- Each generated file should be production-ready and maintainable
- Include appropriate error handling, validation, and documentation
- Follow established patterns and conventions within the codebase
- Ensure code can be easily extended for future requirements

## Communication Guidelines

### When to Ask Questions

- Before creating multiple related files that form a complex unit
- When file organization decisions could significantly impact architecture
- Before implementing cross-cutting concerns that affect multiple components
- When unsure about the scope or depth of implementation needed

### Progress Updates

- Provide brief context about what each generated file accomplishes
- Highlight any important design decisions or trade-offs made
- Indicate what logical component should be built next
- Flag any potential integration points or dependencies for upcoming work