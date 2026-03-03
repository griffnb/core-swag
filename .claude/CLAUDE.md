## This project is a code parsing system that generates OpenAPI specs from code


## ⚠️ CRITICAL: The most important test
`testing/core_models_integration_test.go` :  `TestRealProjectIntegration` and `TestCoreModelsIntegration`
To run on ACTUAL projects, theres 2 make commands to give full true outputs on real projects:
`make test-project-1`
Output: /Users/griffnb/projects/Crowdshield/atlas-go/swag_docs/swagger.json
`make test-project-2`
Output: /Users/griffnb/projects/botbuilders/go-the-schwartz/swag_docs/swagger.json

## ⚠️ CRITICAL: ALWAYS FOLLOW DOCUMENTATION AND PRD
**MANDATORY REQUIREMENT**: Before making ANY changes to this codebase, you MUST:

1. **Maintain consistency**: Any new features, APIs, or changes must align with existing patterns
2. If go docs are missing from a function or package, and you learn something important about it, ADD TO YOUR TODO LIST THAT YOU NEED TO UPDATE THAT GO DOC WITH WHAT YOU LEARNED
3. **VERY IMPORTANT** Do not make large files with lots of functionality.  Group functions together into files that relate them together.  This makes it easier to find grouped functions and their associated tests.  **LARGE FILES ARE BAD**
4. **CHANGE LOG** YOU MUST WITHOUT FAIL DO -> When you try something and it doesnt work, add to the change log ./.agents/change_log.md What you tried, why it didnt work, what you are trying next.


## 🔄 CHECKLIST UPDATE POLICY

**NEVER FORGET**: When you complete any phase, feature, or major milestone:

1. **IMMEDIATELY** update the todo list to mark items as completed
2. **ADD NEW PHASES** to the checklist as they are planned and implemented  
3. **KEEP DOCUMENTATION CURRENT** - the checklist should always reflect the actual project state
4. **UPDATE STATUS** for any infrastructure, integrations, or features that are now working

This ensures the checklist remains an accurate reflection of project progress and helps future development sessions.

**When implementing new features**:
1. Follow established patterns and conventions
2. Update documentation if adding new patterns

**IMPORTANT Before you begin, always launch the context-fetcher sub agent to gather the information required for the task.**