---
description: Perform specialized research tasks for technical decisions, best practices, and technology evaluations.
---

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty).

## Outline

1. **Parse research request**: Extract the research topic, context, and any specific requirements from user input or calling agent.

2. **Determine research scope**:
   - **Technology research**: Best practices, patterns, frameworks for specific technologies
   - **Architecture research**: Design patterns, scalability considerations, integration approaches
   - **Implementation research**: Code patterns, libraries, tools for specific use cases
   - **Domain research**: Industry standards, compliance requirements, domain-specific patterns

3. **Execute research methodology**:
   - **Primary sources**: Official documentation, reputable blogs, academic papers
   - **Secondary sources**: Stack Overflow, GitHub discussions, community forums
   - **Validation**: Cross-reference findings across multiple sources
   - **Current relevance**: Prioritize recent information (last 2-3 years)

4. **Structure research output**:
   - **Decision**: Clear recommendation with rationale
   - **Alternatives considered**: Other options evaluated
   - **Trade-offs**: Pros/cons of chosen approach
   - **Implementation guidance**: How to apply the findings
   - **References**: Sources used for validation

5. **Quality assurance**:
   - Ensure findings align with project constitution
   - Consider project constraints (Go 1.25.4, gRPC, embedded data, etc.)
   - Validate technical feasibility within project architecture

## Research Categories

### Technology Evaluation
- Framework/library assessment
- Language feature analysis
- Tool ecosystem evaluation
- Performance characteristics

### Architecture Patterns
- Design pattern applicability
- Scalability considerations
- Integration approaches
- Security implications

### Implementation Strategies
- Code organization patterns
- Testing approaches
- Deployment strategies
- Monitoring/logging patterns

### Domain-Specific Research
- Industry standards compliance
- Regulatory requirements
- Domain-specific best practices
- Performance benchmarks

## Output Format

**Research Topic:** [Clear topic name]

**Context:** [Why this research is needed]

**Findings:**

1. **Primary Recommendation**
   - **Decision:** [What to use/implement]
   - **Rationale:** [Why this is the best choice]
   - **Evidence:** [Supporting data/sources]

2. **Alternatives Considered**
   - **Option A:** [Description, pros, cons]
   - **Option B:** [Description, pros, cons]

3. **Implementation Guidance**
   - **Prerequisites:** [What needs to be in place]
   - **Steps:** [How to implement]
   - **Potential challenges:** [What to watch out for]

4. **Risks & Mitigations**
   - **Technical risks:** [Potential issues]
   - **Mitigation strategies:** [How to address]

**References:**
- [Source 1]: [Brief description, date]
- [Source 2]: [Brief description, date]

**Validation:** [How findings were validated across sources]

## Quality Standards

- **Objectivity**: Present balanced view of options
- **Actionability**: Provide specific, implementable recommendations
- **Constitution alignment**: Ensure recommendations don't violate project principles
- **Current relevance**: Use recent sources and current best practices
- **Technical accuracy**: Validate technical claims with evidence

Context: $ARGUMENTS