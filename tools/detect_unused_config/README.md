# Configuration Auditing System - README

## Quick Reference

### Commands

```bash
# Quick audit (console output)
make audit-config

# Detailed report (saves to file)
make audit-config-report

# Complete health check
make config-health

# Quick shell script check
./scripts/check_unused_fields.sh
```

### Files

| File | Purpose |
|------|---------|
| `tools/detect_unused_config/main.go` | Static analyzer (AST-based) |
| `scripts/check_unused_fields.sh` | Quick grep-based check |
| `.github/workflows/config-audit.yml` | CI/CD integration |
| `docs/config-auditing.md` | Full documentation |
| `docs/unused-config-detection.md` | Design doc |

---

## What It Does

Systematically detects configuration fields that are:
- ✅ Defined in `pkg/config/types.go`
- ❌ Never accessed in the codebase

**Example output:**
```
❌ Unused Fields: 12 (5.4%)
- DatabaseProviderConfig.Insecure (0 accesses)
- MemoryConfig.Summarization (0 accesses)
...
```

---

## How It Works

1. **Parse:** Extracts all config struct fields using Go AST
2. **Scan:** Searches entire codebase for field accesses
3. **Report:** Categorizes fields and generates recommendations

---

## When to Use

- **During development:** Before committing new config fields
- **In CI:** Automatically on config changes (already configured!)
- **Quarterly:** Manual audits for major cleanups
- **Before releases:** Ensure config is clean

---

## Current Status

Initial audit results (as of implementation):
- **Total fields:** 224
- **Unused:** 12 (5.4%)
- **Lightly used:** 39 (17.4%)
- **Well used:** 173 (77.2%)

### Found Unused Fields

1. `A2AProviderConfig.ContactEmail`
2. `DatabaseProviderConfig.Insecure`
3. `DocumentStoreConfig.AdditionalExcludes`
4. `MemoryConfig.Summarization`
5. `MemoryConfig.SummarizationThreshold`
6. `ReasoningConfig.QualityThreshold`
7. `SearchConfig.MaxContextLength`
8. `SearchConfig.NormalizeWhitespace`
9. `SessionSQLConfig.SSLMode`
10. `TaskSQLConfig.SSLMode`
11. *(+ 2 more)*

---

## CI/CD Integration

### GitHub Actions

Workflow automatically runs on:
- PRs modifying `pkg/config/types.go`
- Pushes to `main`
- Manual trigger

### PR Comments

Bot automatically comments with:
- Summary of unused fields
- Link to full report artifact
- Non-blocking (informational only)

---

## Example Workflow

```bash
# 1. Run audit
$ make audit-config
❌ Found 12 unused fields

# 2. Review report
$ cat config-audit-report.md

# 3. Remove unused field
# Edit pkg/config/types.go

# 4. Verify
$ make audit-config
✅ Found 11 unused fields (one less!)

# 5. Commit
$ git add pkg/config/types.go
$ git commit -m "Remove unused DatabaseProviderConfig.Insecure field"
```

---

## Documentation

- **Full guide:** `docs/config-auditing.md`
- **Design doc:** `docs/unused-config-detection.md`
- **Implementation:** `tools/detect_unused_config/main.go`

---

## Success Story

This tool would have immediately caught the PromptConfig cleanup:

```bash
$ make audit-config

❌ Unused Fields in PromptConfig:
- Instructions (0 accesses)
- Template (0 accesses)
- Variables (0 accesses)
- FullTemplate (0 accesses)
... (8 more)

Result: 12/16 fields unused (75% waste!)
```

**That's exactly what we cleaned up manually.** This tool automates the discovery! 🎯

---

## Maintenance

### Keep the Analyzer Updated

If config structure changes significantly:
1. Update `tools/detect_unused_config/main.go` if needed
2. Test on current codebase: `make audit-config`
3. Update documentation if detection logic changes

### Regular Audits

```bash
# Quarterly or before releases
make audit-config-report

# Review findings
# Create cleanup issues
# Track over time
```

---

## Benefits

1. **Automated Detection** - No manual code review needed
2. **Early Warning** - Catches issues before they accumulate
3. **Clean API** - Maintains honest configuration surface
4. **Developer Productivity** - Less cognitive overhead
5. **Documentation Accuracy** - Config matches reality

---

## Next Steps

1. ✅ Tool implemented and tested
2. ✅ Makefile integration complete
3. ✅ CI/CD workflow added
4. ✅ Documentation created
5. ⏭️ Review current unused fields
6. ⏭️ Create cleanup issues
7. ⏭️ Schedule quarterly audits

---

**Questions?** See `docs/config-auditing.md` for comprehensive guide.

