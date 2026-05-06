#!/usr/bin/env bash

pdc_git_value() {
	local fallback="$1"
	shift

	git "$@" 2>/dev/null || printf '%s\n' "$fallback"
}

pdc_version() {
	pdc_git_value dev describe --tags --always --dirty
}

pdc_commit() {
	pdc_git_value unknown rev-parse --short HEAD
}

pdc_branch() {
	pdc_git_value unknown rev-parse --abbrev-ref HEAD
}

pdc_build_time() {
	date -u +%Y-%m-%dT%H:%M:%SZ
}

pdc_build_by() {
	whoami
}

pdc_ldflags() {
	local module="${MODULE:?MODULE is required}"

	printf "%s" "-s -w"
	printf " -X '%s/internal/version.Version=%s'" "$module" "$(pdc_version)"
	printf " -X '%s/internal/version.Commit=%s'" "$module" "$(pdc_commit)"
	printf " -X '%s/internal/version.Branch=%s'" "$module" "$(pdc_branch)"
	printf " -X '%s/internal/version.BuildTime=%s'" "$module" "$(pdc_build_time)"
	printf " -X '%s/internal/version.BuildBy=%s'" "$module" "$(pdc_build_by)"
}

pdc_binary_path() {
	local dist_dir="${DIST_DIR:?DIST_DIR is required}"
	local binary_name="${BINARY_NAME:?BINARY_NAME is required}"
	local goos="${GOOS:?GOOS is required}"
	local goarch="${GOARCH:?GOARCH is required}"

	printf "%s/%s-%s-%s\n" "$dist_dir" "$binary_name" "$goos" "$goarch"
}
