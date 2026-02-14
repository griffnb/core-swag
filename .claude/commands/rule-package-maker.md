---
name: Package Rule Maker
description: Use to build rules for packages

---

Create a folder inside of rules for the given package, it should follow the package path, i.e.
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
