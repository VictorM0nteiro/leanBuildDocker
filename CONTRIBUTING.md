# Contribuindo

Projeto pequeno, mantenedor solo. Issues e PRs são bem-vindos.

## Setup local

```bash
git clone https://github.com/VictorM0nteiro/leanBuildDocker
cd leanBuildDocker
go test ./...
```

Pré-requisitos: Go 1.23+. Para `--validate`, Docker Desktop rodando.

## Estrutura

| Diretório | Responsabilidade |
|---|---|
| `cmd/lbd/` | CLI (cobra) — orquestrador "burro" |
| `internal/types/` | Contratos `ProjectInfo` e `BuildPlan` |
| `internal/scanner/` | Inventário de arquivos do projeto |
| `internal/detector/` | Identifica linguagem |
| `internal/analyzer/golang/` | Analyzer Go (lê go.mod, detecta CGO) |
| `internal/planner/` | Regras de decisão (alpine vs Debian, etc.) |
| `internal/renderer/` | Templates `text/template` embarcados |
| `internal/knowledge/` | KB de pacotes de sistema (YAML) |
| `internal/validator/` | `--validate` — wrappers de `docker` e `hadolint` |
| `internal/doctor/` | Diagnóstico estático |
| `testdata/projects/` | Fixtures dos analyzers/planners |

Regra de ouro: **um estágio nunca importa o pacote de outro estágio**. Comunicam-se só via `internal/types`.

## Adicionar uma entrada na Knowledge Base

Edite [internal/knowledge/packages.yaml](internal/knowledge/packages.yaml):

```yaml
- go_path: github.com/lib/pq
  requires_cgo: false
  build:
    alpine: []
    debian: []
  runtime:
    alpine: [ca-certificates]
    debian: []
```

Os nomes dos pacotes diferem entre Alpine e Debian — preencha ambos quando aplicável.

## Adicionar um fixture de teste

```bash
mkdir -p testdata/projects/<nome>
echo "module example.com/<nome>" > testdata/projects/<nome>/go.mod
# adicione main.go, dependências, etc.
```

Depois acrescente um caso em `internal/analyzer/golang/analyzer_test.go`.

## Estilo

- `gofmt` + `go vet` limpos antes de PR.
- Wrap erros com `fmt.Errorf("contexto: %w", err)`.
- Comentários só quando o *porquê* não é óbvio do código.
- Sem dependências novas sem justificativa em uma linha de PR.
