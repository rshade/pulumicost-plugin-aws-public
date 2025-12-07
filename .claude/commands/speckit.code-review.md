---
description: Perform automated code review with best practices validation and fix suggestions.
---

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty).

## Outline

1. **Parse review request**: Extract target files/directories, severity level, and any specific focus areas from user input.

2. **Determine review scope**:
   - **Target**: Specific files, directories, or entire codebase
   - **Language**: Auto-detect or specified (Go, JavaScript/TypeScript, Python)
   - **Severity**: low, medium, high (default: medium)
   - **Focus areas**: General best practices, security, performance, maintainability

3. **Execute comprehensive review**:
   - **Static analysis**: Code structure, patterns, anti-patterns
   - **Best practices**: Language-specific conventions and idioms
   - **Security review**: Common vulnerabilities and secure coding practices
   - **Performance review**: Inefficient patterns, resource usage
   - **Maintainability**: Code clarity, documentation, testability

4. **Generate findings report**:
   - **Categorized issues**: By severity and type
   - **Specific locations**: File and line references
   - **Clear descriptions**: What the issue is and why it matters
   - **Fix suggestions**: Concrete recommendations for improvement
   - **Rationale**: Why the suggestion improves code quality

5. **Quality assurance**:
   - Validate findings against project constitution
   - Ensure suggestions align with established patterns
   - Consider project constraints and architecture

## Review Categories

### Code Quality
- **Structure**: Proper organization, separation of concerns
- **Naming**: Clear, consistent naming conventions
- **Documentation**: Adequate comments and documentation
- **Complexity**: Avoid overly complex functions/methods

### Language-Specific Best Practices
- **Go**: Error handling, defer usage, interface design, goroutine safety
- **JavaScript/TypeScript**: Async patterns, type safety, module organization
- **Python**: Exception handling, context managers, type hints

### Security
- **Input validation**: Proper sanitization and validation
- **Resource handling**: Safe file/database/network operations
- **Authentication/Authorization**: Secure access patterns
- **Data exposure**: Prevent sensitive data leakage

### Performance
- **Algorithm efficiency**: Appropriate data structures and algorithms
- **Resource usage**: Memory, CPU, and I/O optimization
- **Concurrency**: Proper synchronization and deadlock prevention
- **Caching**: Effective use of caching strategies

### Testing & Maintainability
- **Test coverage**: Adequate test coverage for critical paths
- **Code duplication**: DRY principle adherence
- **Dependencies**: Appropriate dependency management
- **Future extensibility**: Design for change

## Output Format

**Code Review Report**

**Target:** [Files/directories reviewed]
**Language:** [Detected or specified language]
**Severity Level:** [low/medium/high]
**Review Date:** [Current date]

## Summary
- **Total Issues Found:** [count]
- **Critical Issues:** [count]
- **High Priority:** [count]
- **Medium Priority:** [count]
- **Low Priority:** [count]

## Detailed Findings

### 🔴 Critical Issues
1. **[File:Line]** Issue description
   - **Impact:** [Why this is critical]
   - **Fix:** [Specific recommendation]
   - **Code example:** [Before/after if applicable]

### 🟠 High Priority Issues
1. **[File:Line]** Issue description
   - **Impact:** [Why this matters]
   - **Fix:** [Specific recommendation]
   - **Best practice:** [Reference to standard practice]

### 🟡 Medium Priority Issues
1. **[File:Line]** Issue description
   - **Impact:** [Why this should be addressed]
   - **Fix:** [Specific recommendation]
   - **Improvement:** [Expected benefit]

### 🟢 Low Priority Issues
1. **[File:Line]** Issue description
   - **Impact:** [Minor improvement opportunity]
   - **Fix:** [Optional recommendation]
   - **Convention:** [Reference to coding standard]

## Constitution Compliance
- ✅ **Aligned:** [Aspects that follow project constitution]
- ⚠️ **Concerns:** [Potential constitution conflicts]
- 💡 **Opportunities:** [Ways to better align with constitution]

## Recommendations
1. **Immediate Actions:** [Critical fixes to implement now]
2. **Short-term:** [High priority improvements]
3. **Long-term:** [Quality of life improvements]
4. **Preventive:** [Practices to avoid similar issues]

## Quality Metrics
- **Code Coverage:** [If testable code is present]
- **Complexity Score:** [Relative complexity assessment]
- **Maintainability Index:** [Estimated maintainability]
- **Security Score:** [Security posture assessment]

**Review completed by:** speckit.code-review agent
**Next steps:** [Suggested follow-up actions]

## Context
$ARGUMENTS