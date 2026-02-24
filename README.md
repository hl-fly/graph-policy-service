# Graph Policy Service

Um serviço de inferência de políticas baseado em grafos DOT (Graphviz format) que executa fluxos de decisão complexos em tempo real.

## O que o projeto faz?

O **Graph Policy Service** permite definir políticas complexas usando grafos (modo DOT do Graphviz) e executar inferências baseadas em dados de entrada. 

Exemplos de uso:
- **Análise de risco**: Avaliar pontuação de crédito, idade e histórico
- **Aprovação de pedidos**: Determinar fluxo de aprovação baseado em regras
- **Segmentação de usuários**: Classificar usuários em categorias baseado em critérios
- **Fluxos de decisão**: Qualquer lógica que envolva múltiplas etapas e condições

### Como funciona?

1. Você envia um grafo DOT descrevendo o fluxo de decisão
2. Envia dados de entrada
3. O serviço navega pelo grafo aplicando condições e coletando resultados
4. Retorna os dados de saída com as decisões tomadas

---

## Rotas Disponíveis

### POST `/infer`

Executa uma inferência baseada em um grafo e dados de entrada.

**Request:**
```bash
curl -X POST http://localhost:8080/infer \
  -H "Content-Type: application/json" \
  -d '{
    "policy_dot": "digraph Policy { start [result=\"\"]; approved [result=\"approved=true,segment=prime\"]; rejected [result=\"approved=false\"]; review [result=\"approved=false,segment=manual\"]; start -> approved [cond=\"age>=18 && score>700\"]; start -> review [cond=\"age>=18 && score<=700\"]; start -> rejected [cond=\"age<18\"]; }",
    "input": {
      "age": 25,
      "score": 720
    }
  }'
```

**Response (200 OK):**
```json
{
  "output": {
    "age": 25,
    "score": 720,
    "approved": true,
    "segment": "prime"
  }
}
```

**Explicação do request:**
- `policy_dot`: Grafo em formato DOT que define:
  - `start`: Nó inicial (sem resultado)
  - `approved`: Rota quando `age>=18 && score>700` → aprova como "prime"
  - `review`: Rota quando `age>=18 && score<=700` → fica em review manual
  - `rejected`: Rota quando `age<18` → rejeita
  
- `input`: Dados de entrada para avaliar
  - `age`: 25 anos
  - `score`: 720 pontos

- `output`: Resultado final com todas as variáveis de entrada + decisões tomadas

---

## Como executar

### Opção 1: Usando Makefile (mais simples)

#### Primeiro setup
```bash
make deps
```

#### Rodar testes
```bash
make test
```

#### Rodar com cobertura
```bash
make coverage
```

#### Rodar benchmarks
```bash
make bench
```

#### Iniciar a aplicação
```bash
make run
```

A aplicação estará disponível em `http://localhost:8080`

### Opção 2: Docker Compose

#### Subir o serviço
```bash
make docker-up
```

A aplicação estará disponível em `http://localhost:8080`

#### Parar o serviço
```bash
make docker-down
```

#### Sem Makefile
```bash
# Subir
docker compose up --build

# Parar
docker compose down
```

---

## Arquitetura do Projeto

```
graph-policy-service/
├── cmd/
│   └── api/
│       └── main.go                 # Entry point da aplicação
├── internal/
│   ├── config/
│   │   ├── config.go               # Carregamento de configurações
│   │   └── field.go                # Definição de campos de config
│   ├── domain/
│   │   ├── entity/
│   │   │   └── policy.go           # Entidade Policy (estrutura de dados do grafo)
│   │   └── service/
│   │       └── inferenceservice/
│   │           ├── inferenceservice.go      # Lógica de inferência
│   │           └── inferenceservice_test.go # Testes da inferência
│   └── server/
│       ├── server.go               # Configuração do servidor HTTP
│       ├── handler/
│       │   ├── inferencehandler.go       # Handler da rota /infer
│       │   └── inferencehandler_test.go  # Testes do handler
│       ├── model/
│       │   ├── inference.go        # Modelos de request/response
│       │   └── inference_test.go   # Testes dos modelos
│       └── route/
│           └── route.go            # Definição das rotas
├── Dockerfile                      # Build da imagem Docker
├── docker-compose.yml              # Configuração do Docker Compose
├── Makefile                        # Automação de tasks
├── go.mod                          # Dependências do projeto
└── README.md                       # Este arquivo
```

### Descrição dos diretórios

| Diretório | Responsabilidade |
|-----------|------------------|
| **cmd/** | Pontos de entrada da aplicação |
| **internal/config/** | Carregamento e validação de configurações (variáveis de ambiente) |
| **internal/domain/entity/** | Entidades do domínio (Policy representa o grafo) |
| **internal/domain/service/** | Lógica de negócio (inferência de políticas) |
| **internal/server/** | Infraestrutura HTTP (server, handlers, rotas) |

---

## Stack Técnico

### Chi Router

Usamos [chi](https://github.com/go-chi/chi) para roteamento HTTP por ser:
- Leve e rápido
- Suporte nativo a middleware
- Interface clara e intuitiva
- Compatível com `http.Handler`

### Middlewares

```go
r.Use(middleware.RequestID)      // Adiciona ID único por requisição
r.Use(middleware.Logger)         // Log automático de requisições
r.Use(middleware.Recoverer)      // Recuperação de panics
```

**Detalhes:**
- **RequestID**: Idenifica cada requisição com um UUID. Útil para rastreamento e debugging. Acessível via `middleware.GetReqID(r.Context())`
- **Logger**: Log automático do método HTTP, URL, status code e latência
- **Recoverer**: Captura e recupera de panics, retornando 500 em vez de derrubar a aplicação

### Cache Local (golang-lru)

O serviço usa cache local com [golang-lru](https://github.com/hashicorp/golang-lru) para:

**Por que cache local?**
- Política é imutável (seu hash DOT determina o resultado)
- Não precisa sobreviver a restarts (dados temporários)
- Melhor performance: evita reparse do DOT em cada requisição
- Tamanho controlado: máximo 1000 políticas em memória

**Como funciona:**
1. Calcula hash MD5 do DOT
2. Verifica se está em cache
3. Se não estiver, faz parse e armazena
4. Reutiliza nas próximas requisições com o mesmo DOT

Isso reduz latência de ~100ms para ~1ms em requisições com política cached.

### Expr-lang/expr

Usamos [expr-lang/expr](https://github.com/expr-lang/expr) para avaliar condições nas arestas do grafo.

**Exemplos de condições suportadas:**
```
age>=18 && score>700
user.age > 21 || user.vip == true
score > (age * 10) && status == "active"
len(name) > 5
```

**Benefícios:**
- Parser e compilador de expressões
- Avaliação segura (sem acesso a funções perigosas)
- Performance: compila uma vez, executa múltiplas vezes
- Suporte a tipos complexos (maps, arrays, structs)

### Gographviz

Usamos [gographviz](https://github.com/awalterschulze/gographviz) para fazer parsing e análise de grafos em formato DOT (Graphviz).

**Por que Gographviz?**
- **Padrão aberto**: DOT é um formato amplamente reconhecido e documentado
- **Visual**: Grafos podem ser visualizados em ferramentas como Graphviz, mermaid.js, etc.
- **Parsing robusto**: Gographviz é bem mantida e suporta toda a especificação DOT
- **Análise estruturada**: Converte DOT em estruturas Go que podemos navegar e processar
- **Flexibilidade**: Suporta atributos customizados nos nós e arestas

**Como funciona:**
1. Recebe a string DOT (e.g., `digraph { A [result="x=1"]; B [result="y=2"]; A -> B [cond="x>0"]; }`)
2. Realiza parsing e validação da sintaxe
3. Converte em um grafo estruturado
4. Extrai nós, arestas e seus atributos
5. Mapeamos os atributos customizados (`result`, `cond`) para nossa lógica de negócio

**Pré-processamento:**
Como Graphviz usa `label` e `xlabel` como atributos padrão, fazemos um pequeno pré-processamento para converter:
- `result=` → `label=` (resultado do nó)
- `cond=` → `xlabel=` (condição da aresta)

Isso permite usar uma API padrão mantendo a sintaxe intuitiva para o usuário.

---

## Padrão de Design: Option Pattern

O projeto usa **Option Pattern** para injeção de dependência na inicialização do servidor:

```go
server := server.NewServer(
    server.WithLogger(logger),
    server.WithConfig(cfg),
    server.WithInferenceHandler(inferenceHandler),
)
```

**Benefícios:**
- Argumentos opcionais em Go (sem overloading)
- Ordem flexível dos parâmetros
- Fácil adicionar novas opções sem quebrar código existente
- Valores padrão bem definidos

---

## Variáveis de Ambiente

```bash
SERVER_ADDRESS=:8080  # Endereço e porta do servidor
```

Configure no arquivo `.env` na raiz do projeto.

---

## Testes Unitários e Benchmarks

O projeto possui cobertura completa de testes unitários em três níveis: **model**, **handler** e **service**, além de benchmarks de performance.

### Estrutura dos Testes

Seguimos o padrão **AAA (Arrange, Act, Assert)**:

```go
func TestExecuteInference_ShouldApproveWhenAgeAndScoreValid(t *testing.T) {
    // Arrange - Preparar dados e dependências
    logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
    service := NewInferenceService(logger)
    policy := &entity.Policy{ /* ... */ }

    // Act - Executar a ação
    output, err := service.ExecuteInference(policy, input)

    // Assert - Validar resultados
    require.NoError(t, err)
    assert.Equal(t, true, output["approved"])
}
```

---

## Exemplos Adicionais

### Exemplo 2: Aprovação simples

```json
{
  "policy_dot": "digraph { start [result=\"\"]; end [result=\"approved=true\"]; start -> end; }",
  "input": {}
}
```

**Resposta:**
```json
{
  "output": {
    "approved": true
  }
}
```

### Exemplo 3: Múltiplas condições

```json
{
  "policy_dot": "digraph { start [result=\"\"]; gold [result=\"tier=gold,discount=0.2\"]; silver [result=\"tier=silver,discount=0.1\"]; bronze [result=\"tier=bronze,discount=0\"]; start -> gold [cond=\"purchases > 1000\"]; start -> silver [cond=\"purchases > 500\"]; start -> bronze [cond=\"purchases <= 500\"]; }",
  "input": {
    "purchases": 750
  }
}
```

**Resposta:**
```json
{
  "output": {
    "purchases": 750,
    "tier": "silver",
    "discount": 0.1
  }
}
```

---

## Testes

```bash
# Rodar todos os testes
make test

# Com cobertura
make coverage

# Apenas benchmarks
make bench
```

---

## Troubleshooting

### Erro: `policy_dot` inválido
```json
{
  "result": "invalid policy DOT",
  "details": "..."
}
```
Verifique o formato Graphviz. Use ferramenta online como [Graphviz Online](http://dreampuf.github.io/GraphvizOnline/)

### Erro: Condição com sintaxe inválida
```json
{
  "result": "Error evaluating edge condition",
  "details": "..."
}
```
Verifique a sintaxe da expressão na propriedade `cond` das arestas. Deve ser válida em Go/expr-lang.

---

## Desenvolvimento

Clone o repositório:
```bash
git clone https://github.com/hector-leite/graph-policy-service.git
cd graph-policy-service
```

Setup:
```bash
make deps
```

Rodar em desenvolvimento:
```bash
make run
```

Ou com hot-reload (requer [air](https://github.com/cosmtrek/air)):
```bash
go install github.com/cosmtrek/air@latest
air
```

---

## Licença

MIT
