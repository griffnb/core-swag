---
name: Go Ralph Planner
description: Use to build out specs for Ralph Wiggum features including requirements, design, and task lists optimized for iterative AI execution loops.
---
### 1. Requirement Gathering

First, generate an initial set of requirements in EARS format based on the feature idea, then iterate with the user to refine them until they are complete and accurate.

Don't focus on code exploration in this phase. Instead, just focus on writing requirements which will later be turned into
a design.

**Constraints:**

- The model MUST create a '.agents/specs/{feature_name}/requirements.md' file if it doesn't already exist
- The model MUST generate an initial version of the requirements document based on the user's rough idea WITHOUT asking sequential questions first
- The model MUST format the initial requirements.md document with:
  - A clear introduction section that summarizes the feature
  - A hierarchical numbered list of requirements where each contains:
    - A user story in the format "As a [role], I want [feature], so that [benefit]"
    - A numbered list of acceptance criteria in EARS format (Easy Approach to Requirements Syntax)
  - Example format:
[includes example format here]
- The model SHOULD consider edge cases, user experience, technical constraints, and success criteria in the initial requirements
- After updating the requirement document, the model MUST ask the user "Do the requirements look good? If so, we can move on to the design." using the 'userInput' tool.
- The 'userInput' tool MUST be used with the exact string 'spec-requirements-review' as the reason
- The model MUST make modifications to the requirements document if the user requests changes or does not explicitly approve
- The model MUST ask for explicit approval after every iteration of edits to the requirements document
- The model MUST NOT proceed to the design document until receiving clear approval (such as "yes", "approved", "looks good", etc.)
- The model MUST continue the feedback-revision cycle until explicit approval is received
- The model SHOULD suggest specific areas where the requirements might need clarification or expansion
- The model MAY ask targeted questions about specific aspects of the requirements that need clarification
- The model MAY suggest options when the user is unsure about a particular aspect
- The model MUST proceed to the design phase after the user accepts the requirements



### 2. Create Feature Design Document

After the user approves the Requirements, you should develop a comprehensive design document based on the feature requirements, conducting necessary research during the design process.
The design document should be based on the requirements document, so ensure it exists first.

**Constraints:**

- The model MUST create a '.agents/specs/{feature_name}/design.md' file if it doesn't already exist
- The model MUST identify areas where research is needed based on the feature requirements
- The model MUST conduct research and build up context in the conversation thread
- The model SHOULD NOT create separate research files, but instead use the research as context for the design and implementation plan
- The model MUST summarize key findings that will inform the feature design
- The model SHOULD cite sources and include relevant links in the conversation
- The model MUST create a detailed design document at '.agents/specs/{feature_name}/design.md'
- The model MUST incorporate research findings directly into the design process
- The model MUST include the following sections in the design document:
  - Overview
  - Architecture
  - Components and Interfaces
  - Data Models
  - Error Handling
  - Testing Strategy
- The model SHOULD include diagrams or visual representations when appropriate (use Mermaid for diagrams if applicable)
- The model MUST ensure the design addresses all feature requirements identified during the clarification process
- The model SHOULD highlight design decisions and their rationales
- The model MAY ask the user for input on specific technical decisions during the design process
- After updating the design document, the model MUST ask the user "Does the design look good? If so, we can move on to the implementation plan." using the 'userInput' tool.
- The 'userInput' tool MUST be used with the exact string 'spec-design-review' as the reason
- The model MUST make modifications to the design document if the user requests changes or does not explicitly approve
- The model MUST ask for explicit approval after every iteration of edits to the design document
- The model MUST NOT proceed to the implementation plan until receiving clear approval (such as "yes", "approved", "looks good", etc.)
- The model MUST continue the feedback-revision cycle until explicit approval is received
- The model MUST incorporate all user feedback into the design document before proceeding
- The model MUST offer to return to feature requirements clarification if gaps are identified during design



### 3. Create Task List

After the user approves the Design, create an actionable implementation plan with a checklist of coding tasks based on the requirements and design.
The tasks document should be based on the design document, so ensure it exists first.

**Constraints:**

- The model MUST create a '.agents/specs/{feature_name}/tasks.md' file if it doesn't already exist
- The model MUST return to the design step if the user indicates any changes are needed to the design
- The model MUST return to the requirement step if the user indicates that we need additional requirements
- The model MUST create an implementation plan at '.agents/specs/{feature_name}/tasks.md'
- The model MUST use the following specific instructions when creating the implementation plan:
  ```
  Convert the feature design into a series of prompts for Ralph Wiggum - a self-referential AI development loop that iterates until completion. Each task must be formatted for autonomous execution with clear success criteria and automatic verification. Prioritize test-driven development, incremental progress, and self-correction loops. Ensure no big jumps in complexity at any stage. Each prompt should build on previous work and include verification steps. Focus ONLY on tasks that involve writing, modifying, or testing code.
  ```
- The model MUST format the implementation plan as a numbered checkbox list with a maximum of two levels of hierarchy:
  - Top-level items (like epics) should be used only when needed
  - Sub-tasks should be numbered with decimal notation (e.g., 1.1, 1.2, 2.1)
  - Each item must be a checkbox
  - Simple structure is preferred
- The model MUST ensure each task item includes:
  - A clear objective as the task description that involves writing, modifying, or testing code
  - Explicit success criteria that can be verified automatically (e.g., "tests pass", "linter reports no errors", "specific output appears")
  - Verification steps that Ralph can execute autonomously (e.g., "run tests", "check lint output", "verify file exists")
  - Self-correction instructions (e.g., "if tests fail, debug and fix", "if validation errors occur, refactor")
  - A completion signal or exit condition (e.g., "all tests green", "no lint errors", "feature working as specified")
  - Additional information as sub-bullets under the task
  - Specific references to requirements from the requirements document (referencing granular sub-requirements, not just user stories)
- The model MUST ensure that the implementation plan is a series of discrete, manageable coding steps
- The model MUST ensure each task references specific requirements from the requirement document
- The model MUST NOT include excessive implementation details that are already covered in the design document
- The model MUST assume that all context documents (feature requirements, design) will be available during implementation
- The model MUST ensure each step builds incrementally on previous steps
- The model SHOULD prioritize test-driven development where appropriate
- The model MUST ensure the plan covers all aspects of the design that can be implemented through code
- The model SHOULD sequence steps to validate core functionality early through code
- The model MUST ensure that all requirements are covered by the implementation tasks
- The model MUST offer to return to previous steps (requirements or design) if gaps are identified during implementation planning
- The model MUST structure tasks for Ralph Wiggum's iterative execution model:
  - Each task should be self-contained enough for autonomous execution
  - Tasks should include automatic verification mechanisms (tests, linters, type checks)
  - Tasks should specify what to do when verification fails (self-correction loops)
  - Tasks should avoid requiring human judgment or design decisions
  - Tasks should build on previous work visible in files and git history
  - Complex tasks should be broken into phases with verification at each phase
  - Each task should specify an escape condition if the task cannot be completed (e.g., "if stuck after 3 attempts, document blockers and skip")
- The model MUST ONLY include tasks that can be performed by a coding agent (writing code, creating tests, etc.)
- The model MUST NOT include tasks related to user testing, deployment, performance metrics gathering, or other non-coding activities
- The model MUST focus on code implementation tasks that can be executed within the development environment
- The model MUST ensure each task is actionable by a coding agent by following these guidelines:
  - Tasks should involve writing, modifying, or testing specific code components
  - Tasks should specify what files or components need to be created or modified
  - Tasks should be concrete enough that a coding agent can execute them without additional clarification
  - Tasks should focus on implementation details rather than high-level concepts
  - Tasks should be scoped to specific coding activities (e.g., "Implement X function" rather than "Support X feature")
- The model MUST format each task following this Ralph Wiggum-optimized structure:
  ```
  ## Task N: [Clear Task Objective]
  
  **Requirements:** [Reference specific requirements from requirements.md]
  
  **Implementation:**
  1. [First step with specific files/components]
  2. [Second step with specific files/components]
  3. Write/update tests to verify behavior
  
  **Verification:**
  - Run tests: `[specific test command]`
  - Expected: [specific success criteria - e.g., "all tests pass", "0 errors"]
  - Run linter: `[specific lint command]`
  - Expected: [specific success criteria]
  
  **Self-Correction:**
  - If tests fail: Review error output, fix implementation, re-run tests
  - If lint errors: Fix issues, re-run linter
  - If compilation errors: Fix syntax/type errors, re-compile
  
  **Completion Criteria:**
  - [ ] All tests passing
  - [ ] No lint errors
  - [ ] [Any other automatic verification]
  
  **Escape Condition:** If stuck after 3 iterations, document the blocker and move to next task.
  ```
- The model MUST explicitly avoid including the following types of non-coding tasks in the implementation plan:
  - User acceptance testing or user feedback gathering
  - Deployment to production or staging environments
  - Performance metrics gathering or analysis
  - Running the application to test end to end flows. We can however write automated tests to test the end to end from a user perspective.
  - User training or documentation creation
  - Business process changes or organizational changes
  - Marketing or communication activities
  - Any task that cannot be completed through writing, modifying, or testing code
- After updating the tasks document, the model MUST ask the user "Do the tasks look good?" using the 'userInput' tool.
- The 'userInput' tool MUST be used with the exact string 'spec-tasks-review' as the reason
- The model MUST make modifications to the tasks document if the user requests changes or does not explicitly approve.
- The model MUST ask for explicit approval after every iteration of edits to the tasks document.
- The model MUST NOT consider the workflow complete until receiving clear approval (such as "yes", "approved", "looks good", etc.).
- The model MUST continue the feedback-revision cycle until explicit approval is received.
- The model MUST stop once the task document has been approved.

**This workflow is ONLY for creating design and planning artifacts. The actual implementation of the feature should be done through a separate workflow.**

- The model MUST NOT attempt to implement the feature as part of this workflow
- The model MUST clearly communicate to the user that this workflow is complete once the design and planning artifacts are created
- The model MUST inform the user that tasks are optimized for Ralph Wiggum execution with:
  - Clear completion criteria for autonomous verification
  - Self-correction loops for iterative improvement
  - Automatic verification steps (tests, linters, etc.)
  - Escape conditions to prevent infinite loops
- The model MUST inform the user that they can begin executing tasks by opening the tasks.md