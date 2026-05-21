# AgentLoopOrchestrator

Orquestrador em loop que executa agentes **opencode** periodicamente (a cada 60 segundos) para tarefas automatizadas de manutenção de projeto.

## Como funciona

O `main.go` define uma lista de tarefas (`Task`) com:

| Campo       | Descrição |
|-------------|-----------|
| `Type`      | Tipo da tarefa (ex: `"normal"`) |
| `InitMessage` | Mensagem de inicialização enviada ao agente |
| `Agent`     | Nome do binário do agente (`"opencode"`) |
| `AgentModel` | Modelo de IA usado (`"iproute/deepseek-v4-flash"`) |
| `AgentMode` | Modo de operação (`"build"`) |
| `Path`      | Caminho absoluto do projeto alvo |

A cada 60 segundos, o orquestrador executa:

```bash
opencode run --agent build --model <model> "<init-message>"
```

O projeto é **auto-referencial** — ele orquestra tarefas de manutenção sobre si mesmo (ex: verificar/atualizar o próprio README).

## Tarefas atuais

1. **Checkup do README** — verifica se o README está de acordo com o projeto e o cria se não existir.

## Pré-requisitos

- Go 1.25+
- [opencode](https://opencode.ai) CLI instalado e disponível no `PATH`

## Executar

```bash
go run main.go
```
