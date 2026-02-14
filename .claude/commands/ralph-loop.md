---
description: "Start Ralph Loop in current session"
argument-hint: "PROMPT [--max-iterations N] [--completion-promise TEXT]"
hide-from-slash-command-tool: "true"
---

# Ralph Loop Command

Execute the setup script to initialize the Ralph loop: 

Bash("./.claude/hooks/scripts/setup-ralph-loop.sh" $ARGUMENTS)


the bash script above expects arguments of the prompt youve been asked to do, so if you dont see it below, please add it before calling the tool
<example>
Bash("./.claude/hooks/scripts/setup-ralph-loop.sh \"Complete the tasks in file xyz\")
</example>


Please work on the task. When you try to exit, the Ralph loop will feed the SAME PROMPT back to you for the next iteration. You'll see your previous work in files and git history, allowing you to iterate and improve.

CRITICAL RULE: If a completion promise is set, you may ONLY output it when the statement is completely and unequivocally TRUE. Do not output false promises to escape the loop, even if you think you're stuck or should exit for other reasons. The loop is designed to continue until genuine completion.
