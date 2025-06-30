# NeuroMesh Repository Migration Plan

## Overview
Migrate the orchestrator component from the ZTDP monorepo to a new dedicated NeuroMesh repository with clean branding and structure.

## Migration Strategy

### 1. Pre-Migration Preparation
- [x] Document brand decision (NeuroMesh)
- [ ] Create new GitHub repository: `neuromesh`
- [ ] Plan new directory structure
- [ ] Identify files to migrate

### 2. New Repository Structure
```
neuromesh/
├── README.md                 # NeuroMesh introduction
├── LICENSE                   # MIT License
├── .gitignore               # Go + IDE ignores
├── go.mod                   # module: github.com/your-org/neuromesh
├── go.sum
├── Makefile                 # Build automation
├── docker-compose.yml       # Development environment
├── .github/
│   ├── workflows/           # CI/CD pipelines
│   └── ISSUE_TEMPLATE/      # Bug/feature templates
├── cmd/
│   ├── neuromesh/           # Main NeuroMesh service
│   │   └── main.go
│   └── neuromesh-cli/       # CLI tool
│       └── main.go
├── internal/                # Private application code
│   ├── agents/              # Agent management
│   ├── orchestrator/        # Core orchestration logic
│   ├── web/                 # Web BFF
│   ├── logging/            # Logging infrastructure
│   └── shared/             # Shared utilities
├── pkg/                    # Public API packages
│   ├── neuromesh/          # Public NeuroMesh API
│   └── types/              # Shared types
├── api/                    # API definitions
│   ├── openapi/            # OpenAPI specs
│   └── grpc/               # Protocol buffer definitions
├── docs/                   # Documentation
│   ├── architecture/       # Architecture docs
│   ├── guides/            # User guides
│   └── api/               # API documentation
├── examples/              # Example configurations
├── test/                  # Integration tests
├── scripts/              # Build and deployment scripts
└── deployments/          # Kubernetes/Docker configs
    ├── docker/
    └── k8s/
```

### 3. Files to Migrate from `/orchestrator/`

#### Core Application Code
- `internal/orchestrator/` → `internal/orchestrator/`
- `internal/web/` → `internal/web/`
- `internal/logging/` → `internal/logging/`
- `internal/agentFramework/` → `internal/agents/framework/`
- `internal/agentRegistry/` → `internal/agents/registry/`
- `internal/agents/` → `internal/agents/`
- `internal/ai/` → `internal/ai/`
- `internal/application/` → `internal/application/`
- `internal/shared/` → `internal/shared/`

#### Entry Points
- `cmd/api/main.go` → `cmd/neuromesh/main.go`

#### Configuration
- `go.mod` → update module path
- `docker-compose.yml` → update service names
- `.gitignore` → merge and clean up

#### Documentation
- Select relevant docs from `docs/` → `docs/`
- Create new NeuroMesh README

### 4. Rebranding Changes

#### Module Path
- From: `github.com/ztdp/orchestrator`
- To: `github.com/your-org/neuromesh`

#### Service Names
- `orchestrator` → `neuromesh`
- `api` service → `neuromesh-api`
- Internal references → update to NeuroMesh terminology

#### Comments and Documentation
- Update all comments referencing "orchestrator"
- Rebrand to "NeuroMesh" or "neural mesh"
- Update API documentation

### 5. Migration Steps

#### Step 1: Create New Repository
```bash
# Create new repo on GitHub: neuromesh
# Clone locally
git clone https://github.com/your-org/neuromesh.git
cd neuromesh
```

#### Step 2: Setup Base Structure
- Create directory structure
- Add base files (README, LICENSE, .gitignore)
- Initialize Go module

#### Step 3: Migrate Core Code
- Copy `/orchestrator/internal/` to `neuromesh/internal/`
- Copy `/orchestrator/cmd/` to `neuromesh/cmd/`
- Update import paths
- Update module references

#### Step 4: Clean and Rebrand
- Update all "orchestrator" references to "neuromesh"
- Clean up unused dependencies
- Update configuration files
- Rebrand comments and documentation

#### Step 5: Test Migration
- Ensure all tests pass
- Verify build works
- Test API endpoints
- Validate Docker compose

#### Step 6: Finalization
- Create comprehensive README
- Add examples and documentation
- Setup CI/CD pipelines
- Tag initial release

### 6. Post-Migration Tasks
- [ ] Update ZTDP repo to reference NeuroMesh
- [ ] Archive/deprecate orchestrator in ZTDP
- [ ] Setup GitHub Actions for NeuroMesh
- [ ] Create initial documentation site
- [ ] Announce NeuroMesh project

## Timeline
- **Day 1**: Repository setup and core migration
- **Day 2**: Rebranding and testing
- **Day 3**: Documentation and examples
- **Day 4**: CI/CD and release preparation

## Risk Mitigation
- Keep ZTDP orchestrator as backup during migration
- Test thoroughly before deprecating old version
- Document breaking changes
- Provide migration guide for existing users

---
*Created: June 30, 2025*
*Status: Ready for execution*
