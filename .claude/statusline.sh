#!/bin/bash
# Read JSON input once
input=$(cat)

# Color codes
RESET='\033[0m'
BOLD='\033[1m'
BLUE='\033[34m'
CYAN='\033[36m'
GREEN='\033[32m'
YELLOW='\033[33m'
RED='\033[31m'
MAGENTA='\033[35m'
DIM='\033[2m'

# Helper functions for common extractions
get_model_name() { echo "$input" | jq -r '.model.display_name'; }
get_current_dir() { echo "$input" | jq -r '.workspace.current_dir'; }
get_project_dir() { echo "$input" | jq -r '.workspace.project_dir'; }
get_context_window_size() { echo "$input" | jq -r '.context_window.context_window_size'; }
get_current_usage() { echo "$input" | jq '.context_window.current_usage'; }
get_cost() { echo "$input" | jq -r '.cost.total_cost_usd'; }
get_git_branch() {
    cd "$(echo "$input" | jq -r '.workspace.project_dir')" 2>/dev/null
    git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "no-git"
}

# Extract values using helpers
MODEL=$(get_model_name)
DIR=$(get_current_dir)
CONTEXT_SIZE=$(get_context_window_size)
USAGE=$(get_current_usage)
COST=$(get_cost)
BRANCH=$(get_git_branch)

# Calculate current context usage
if [ "$USAGE" != "null" ]; then
    CURRENT_TOKENS=$(echo "$USAGE" | jq '.input_tokens + .cache_creation_input_tokens + .cache_read_input_tokens')
    PERCENT_USED=$((CURRENT_TOKENS * 100 / CONTEXT_SIZE))
else
    PERCENT_USED=0
fi

# Determine cost color (green if <= $10, transition to red after)
if (( $(echo "$COST > 10" | bc -l) )); then
    COST_COLOR=$RED
elif (( $(echo "$COST > 5" | bc -l) )); then
    COST_COLOR=$YELLOW
else
    COST_COLOR=$GREEN
fi

# Format cost to 2 decimal places
COST_FORMATTED=$(printf "%.2f" "$COST")

# Build status line with colors
echo -e "${BOLD}${BLUE}[$MODEL]${RESET} ${DIM}|${RESET} ${CYAN}📁 ${DIR##*/}${RESET} ${DIM}|${RESET} ${MAGENTA}⎇ $BRANCH${RESET} ${DIM}|${RESET} ${YELLOW}Context: ${PERCENT_USED}%${RESET} ${DIM}|${RESET} ${COST_COLOR}💰 \$$COST_FORMATTED${RESET}"
