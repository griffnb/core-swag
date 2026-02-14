---
name: Rule Maker
description: Use to build rules

---

Review the conversation and what you learned, return a highlight summary of the things you did wrong and the rules you'd want to make to be sure it doesnt happen again.

After i approve the rule creation, you'd follow this guidance:


## Specific Package Learnings

Create a folder inside of rules for the given package if its a package based rule, it should follow the package path, i.e.
`/rules/services/myservice/myservice.md` where the pattern is `/rules/{package}/{path}/{package_name}.md`

Inside this rules file there should be a glob of the path at the top in frontmatter
```md
---
paths:
  - "internal/{path to}/{package name here}/**/*.go"
---
```



Inside this rules file there should be sections

## Overview
{simple overview of what this package is}

## Key Structs/Methods
- {any important structs / methods that are the main use of this package}
- {any entry points}

Should be in the format of: 
- [FunctionNameStructName( any params)](link to code) Short description of what it does

Example:
- [WelcomeEvent(ctx, userAccountID)](../../../../internal/services/account_event/invites.go#L17) - Creates welcome event for new users



## Related Packages
- any other packages that rely on this, i.e. services that use this integration or any important linked packages

## Docs
- [Link To The Doc](here)  // any existing docs/readmes that go into more detail

   
## Related Skills // any related skills to this package following the below format examples
- [Skill(model-conventions)](../../skills/model-conventions) 
- [Skill(model-queries)](../../skills/model-queries) 
- [Skill(model-usage)](../../skills/model-usage)




## General Learnings


Create a file inside of rules for the given learnings if there is a rule already, then append to it.
The name of the file should be relative to the ruleset.

Inside this rules file there should be a glob of the path at the top in frontmatter if it should only be filtered to certain things
```md
---
paths:
  - "internal/{path to}/{package name here}/**/*.go"
---
```


Inside this rules file there should be sections

## Overview
{simple overview what the rule is}

## MUSTs
- {List of rules that are requirements in the format of **AGENT MUST** xxxxxxxxxxx}

## MUST NOTS

- {List of rules that are requirements in the format of **AGENT MUST** xxxxxxxxxxx}


If examples are needed in the musts or must nots, use linked formats:
Example:
- **MUST USE ZEROLOG** [ZeroLogger(ctx, someParam, someMessage string)](../../../../internal/services/account_event/invites.go#L17)event for new users

## Related Skills // any related skills to this rule if they exist or give more detail following this pattern
- [Skill(model-conventions)](../../skills/model-conventions) 
- [Skill(model-queries)](../../skills/model-queries) 
- [Skill(model-usage)](../../skills/model-usage)
