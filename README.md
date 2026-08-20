# AWS Step Functions Activity Worker

Exemplo de integração entre **AWS Step Functions Activities** e um **worker local desenvolvido em Go**.

O projeto demonstra como uma máquina de estado do AWS Step Functions pode delegar uma etapa para um processo externo, utilizando uma **Step Functions Activity**. O worker Go executa localmente, faz polling da Activity, processa a tarefa e devolve o resultado para o Step Functions.

As Activities permitem que o trabalho de uma tarefa seja executado por um processo externo ao Step Functions, que pode estar rodando em EC2, ECS, Kubernetes, ambiente on-premises ou até mesmo localmente durante o desenvolvimento.

## Arquitetura

```text
                         AWS
┌───────────────────────────────────────────────────────┐
│                                                       │
│  ┌───────────────────────────────────────────────┐    │
│  │             AWS Step Functions                │    │
│  │                                               │    │
│  │  Parse Input                                  │    │
│  │       │                                       │    │
│  │       ▼                                       │    │
│  │  Generate Processing Dates                    │    │
│  │       │                                       │    │
│  │       ▼                                       │    │
│  │  Map / Process Data                           │    │
│  │       │                                       │    │
│  │       ▼                                       │    │
│  │  Consolidate Results                          │    │
│  │       │                                       │    │
│  │       ▼                                       │    │
│  │  Send To Approver                             │    │
│  │       │                                       │    │
│  └───────┼───────────────────────────────────────┘    │
│          │                                            │
│          │ GetActivityTask                            │
│          │ SendTaskSuccess / SendTaskFailure          │
│          ▼                                            │
│  ┌───────────────────────────────────────────────┐    │
│  │              Step Functions Activity          │    │
│  │              approver-worker                  │    │
│  └───────────────────────┬───────────────────────┘    │
│                          │                            │
└──────────────────────────┼────────────────────────────┘
                           │
                           │ AWS SDK
                           ▼
                  ┌──────────────────┐
                  │   Go Worker      │
                  │                  │
                  │ GetActivityTask  │
                  │       │          │
                  │       ▼          │
                  │  Process Task    │
                  │       │          │
                  │       ▼          │
                  │ SendTaskSuccess  │
                  │       ou         │
                  │ SendTaskFailure  │
                  └──────────────────┘
                           │
                           ▼
                       Máquina
                        local
```

O fluxo utiliza as APIs `GetActivityTask`, `SendTaskSuccess` e `SendTaskFailure` da AWS Step Functions.

## Componentes

### Terraform

A infraestrutura AWS é criada usando Terraform.

Os principais recursos são:

* AWS Step Functions Activity
* AWS Step Functions State Machine
* IAM Role para o Step Functions
* IAM Policy
* Outputs contendo os ARNs necessários para executar o worker

A Activity é criada como:

```hcl
resource "aws_sfn_activity" "approver" {
  name = "${var.project_name}-approver-worker"
}
```

A State Machine utiliza essa Activity como recurso de um estado `Task`.

### Go Activity Worker

O worker está localizado em:

```text
app/
├── go.mod
├── go.sum
└── main.go
```

O programa utiliza o AWS SDK for Go v2 e o cliente `sfn` para comunicação com o AWS Step Functions.

O worker:

1. Obtém o ARN da Activity através da variável `APPROVER_WORKER_ARN`.
2. Cria um cliente do Step Functions.
3. Executa `GetActivityTask`.
4. Aguarda uma tarefa.
5. Desserializa o input recebido.
6. Processa a tarefa.
7. Envia `SendTaskSuccess` em caso de sucesso.
8. Envia `SendTaskFailure` em caso de erro.

## Step Functions

A State Machine é criada pelo Terraform e utiliza:

```text
QueryLanguage = JSONata
```

O fluxo contém os seguintes estados:

```text
Parse Input Data
       │
       ▼
Generate Next Processing Dates
       │
       ▼
Proccess Data
       │
       ├── Wait For Start Datetime
       │
       └── Init Hourly Processing
       │
       ▼
Consolidate Results
       │
       ▼
Send To Approver
```

### Parse Input Data

O workflow recebe uma data através do campo:

```json
{
  "datetime": "2026-08-19"
}
```

Essa informação é convertida para a data de processamento utilizando JSONata.

### Generate Next Processing Dates

O projeto utiliza uma expressão JSONata definida no Terraform para gerar uma série de horários de processamento.

A expressão gera entradas contendo:

```json
{
  "src_ptt": "...",
  "dst_ptt": "...",
  "exec": "..."
}
```

São geradas 21 entradas, correspondentes ao intervalo `[0..20]` utilizado na expressão JSONata.

### Process Data

O estado `Proccess Data` utiliza um `Map` para processar as entradas geradas anteriormente.

Cada item passa primeiro pelo estado:

```text
Wait For Start Datetime
```

e aguarda até o timestamp definido para aquela execução.

Em seguida, `Init Hourly Processing` prepara os dados:

```json
{
  "data_movimento": "...",
  "src_partition": "...",
  "dst_partition": "...",
  "job_run_id": "..."
}
```

O `job_run_id` é gerado através de `$uuid()`.

### Consolidate Results

Após o processamento do `Map`, o workflow consolida o resultado e gera um novo identificador de execução.

### Send To Approver

Finalmente, a State Machine envia os dados para a Activity:

```text
Send To Approver
       │
       ▼
AWS Step Functions Activity
       │
       ▼
Go Worker
```

O recurso da Activity é obtido diretamente do Terraform:

```hcl
Resource = aws_sfn_activity.approver.id
```

## Pré-requisitos

Para executar o projeto, é necessário ter instalado:

* AWS CLI
* Terraform >= 1.3
* Go
* `jq`
* Make

Também é necessário possuir credenciais AWS com permissões suficientes para criar os recursos Terraform.

O projeto utiliza o provider AWS >= 5.0.

## Configuração AWS

Configure suas credenciais AWS através de um dos mecanismos suportados pelo AWS SDK/CLI.

Por exemplo:

```bash
aws configure
```

Verifique se as credenciais estão funcionando:

```bash
aws sts get-caller-identity
```

A aplicação Go utiliza a configuração padrão do AWS SDK e executa na região:

```text
sa-east-1
```

## Deploy da infraestrutura

Inicialize o Terraform:

```bash
terraform init
```

Valide a configuração:

```bash
terraform validate
```

Formate os arquivos:

```bash
terraform fmt
```

Aplique a infraestrutura:

```bash
terraform apply
```

Ou utilize o target disponibilizado no `Makefile`:

```bash
make infra-deploy
```

O target executa:

```text
terraform fmt
terraform validate
terraform apply
```

## Outputs

Após o deploy, o Terraform disponibiliza dois outputs:

```text
approver_activity_arn
state_machine_arn
```

Eles correspondem, respectivamente, ao ARN da Activity e ao ARN da State Machine.

Para consultar:

```bash
terraform output
```

Ou:

```bash
terraform output -json
```

## Executando o Activity Worker

Depois de criar a infraestrutura, o worker precisa receber o ARN da Activity através da variável:

```bash
APPROVER_WORKER_ARN
```

O `Makefile` já automatiza isso:

```bash
make go
```

Internamente, o comando obtém o ARN através do Terraform:

```bash
terraform output -json | jq -r '.approver_activity_arn.value'
```

e executa:

```bash
cd app/
go run main.go
```

Também é possível executar manualmente:

```bash
export APPROVER_WORKER_ARN=$(terraform output -raw approver_activity_arn)

cd app
go run main.go
```

## Iniciando uma execução

Com a infraestrutura criada e o worker em execução, inicie uma execução da State Machine:

```bash
make start-execution
```

O comando executado pelo `Makefile` é equivalente a:

```bash
aws stepfunctions start-execution \
  --state-machine-arn "$STATE_MACHINE_ARN" \
  --input '{"datetime":"2026-08-19"}'
```

Também é possível executar manualmente:

```bash
STATE_MACHINE_ARN=$(terraform output -raw state_machine_arn)

aws stepfunctions start-execution \
  --state-machine-arn "$STATE_MACHINE_ARN" \
  --input '{"datetime":"2026-08-19"}'
```

## Worker Go

O worker utiliza o AWS SDK for Go v2:

```go
sfnClient := sfn.NewFromConfig(cfg)
```

e realiza o polling da Activity:

```go
output, err := sfnClient.GetActivityTask(ctx, &sfn.GetActivityTaskInput{
    ActivityArn: aws.String(activityArn),
    WorkerName:  aws.String(workerName),
})
```

Quando uma tarefa é recebida, o input é convertido para:

```go
type ApproverWorkerInput struct {
    JobRunId          string `json:"job_run_id"`
    ProcessedPartition string `json:"processed_partition"`
}
```

Após o processamento, o worker retorna informações como:

```json
{
  "task_token": "...",
  "has_error": false,
  "approved_by": "hostname",
  "native_os": "linux",
  "native_arch": "amd64",
  "job_run_id": "...",
  "processed_partition": "..."
}
```

O hostname da máquina que executou o worker é utilizado como `approved_by`, permitindo identificar o processo que efetivamente executou a Activity.

## Sucesso e falha

Quando o processamento é concluído com sucesso, o worker utiliza:

```text
SendTaskSuccess
```

e envia o resultado serializado em JSON.

Em caso de erro:

```text
SendTaskFailure
```

é utilizado com o mesmo `taskToken`.

Essa associação através do `taskToken` é o mecanismo utilizado pelo Step Functions para relacionar o resultado produzido pelo worker à tarefa específica da execução.

## Estrutura do projeto

```text
step-function-activity/
│
├── app/
│   ├── go.mod
│   ├── go.sum
│   └── main.go
│
├── .gitignore
├── Makefile
├── iam.tf
├── main.tf
├── outputs.tf
├── states.tf
└── variables.tf
```

### Arquivos Terraform

| Arquivo        | Responsabilidade                                |
| -------------- | ----------------------------------------------- |
| `main.tf`      | Configuração do Terraform e provider AWS        |
| `iam.tf`       | IAM Role e Policy utilizados pela State Machine |
| `states.tf`    | Activity e State Machine                        |
| `variables.tf` | Variáveis Terraform                             |
| `outputs.tf`   | Outputs dos ARNs                                |
| `Makefile`     | Automação de deploy, worker e execução          |

O projeto possui atualmente uma variável Terraform principal:

```hcl
variable "project_name" {
  type    = string
  default = "sfn-activity"
}
```

## IAM

A State Machine utiliza uma IAM Role que pode ser assumida pelo serviço:

```text
states.amazonaws.com
```

A policy atualmente configurada concede:

```text
s3:GetObject
s3:ListBucket
```

sobre os recursos configurados pela policy.

> **Atenção:** para ambientes reais, recomenda-se restringir os recursos da policy aos buckets e objetos efetivamente necessários, em vez de utilizar `"resources = ["*"]`.

## Execução local

O principal objetivo do exemplo é permitir que o worker seja executado fora da infraestrutura do Step Functions.

Isso significa que o worker pode estar rodando em:

* notebook de desenvolvimento;
* servidor local;
* VM;
* EC2;
* ECS;
* Kubernetes;
* ambiente on-premises.

Desde que consiga acessar a API do AWS Step Functions e possua credenciais AWS adequadas.

Esse modelo é justamente uma das características das Step Functions Activities: o worker pode ser executado fora do ambiente gerenciado pelo Step Functions.

## Fluxo completo de execução

Uma execução típica ocorre da seguinte forma:

```text
1. terraform apply
        │
        ▼
2. Activity criada
        │
        ▼
3. State Machine criada
        │
        ▼
4. Go Worker iniciado
        │
        ▼
5. start-execution
        │
        ▼
6. Step Functions processa os estados
        │
        ▼
7. Step Functions chega em "Send To Approver"
        │
        ▼
8. Activity recebe uma tarefa
        │
        ▼
9. Go Worker executa GetActivityTask
        │
        ▼
10. Worker recebe taskToken + input
        │
        ▼
11. Worker processa a tarefa
        │
        ▼
12. SendTaskSuccess / SendTaskFailure
        │
        ▼
13. Step Functions continua a execução
```

## Limitações e pontos de atenção

### Timeout do worker

O worker cria um contexto com timeout de 65 segundos:

```go
context.WithTimeout(context.Background(), 65*time.Second)
```

Enquanto o `GetActivityTask` pode aguardar até aproximadamente 60 segundos por uma tarefa.

Como o mesmo contexto é utilizado posteriormente para processar e responder à tarefa, atividades que sejam recebidas próximo do limite do timeout podem ficar sem tempo suficiente para executar o processamento e chamar `SendTaskSuccess`.

Para um worker de produção, recomenda-se separar o contexto de polling do contexto utilizado para processar a tarefa e implementar uma estratégia explícita de timeout/retry.

### Retry

O código atualmente possui um TODO:

```go
// TODO: Add retry
```

Portanto, falhas de comunicação com o Step Functions não possuem ainda uma política de retry implementada no worker.

### Heartbeat

O exemplo não implementa `SendTaskHeartbeat`.

Para Activities de longa duração, o worker deve enviar heartbeats de acordo com o `HeartbeatSeconds` configurado na tarefa, quando aplicável. A AWS recomenda o uso de heartbeats para manter atividades longas ativas.

### Concorrência

O worker atual possui um único loop de polling e processa a tarefa diretamente.

Para cenários de maior throughput, uma implementação de produção pode utilizar:

```text
pollers
   │
   ▼
queue
   │
   ├── worker 1
   ├── worker 2
   ├── worker 3
   └── worker N
```

Isso permite separar a aquisição das tarefas do processamento.

## Tecnologias

* **AWS Step Functions**
* **AWS Step Functions Activities**
* **AWS SDK for Go v2**
* **Go**
* **Terraform**
* **AWS IAM**
* **JSONata**
* **Amazon States Language**

## Referências

* [AWS Step Functions Activities](https://docs.aws.amazon.com/step-functions/latest/dg/concepts-activities.html)
* [AWS Step Functions API](https://docs.aws.amazon.com/step-functions/latest/apireference/API_Operations.html)
* [AWS SDK for Go v2](https://aws.github.io/aws-sdk-go-v2/)
* [Terraform AWS Provider](https://registry.terraform.io/providers/hashicorp/aws/latest)

## Licença

Este projeto não possui uma licença explícita definida no repositório atualmente.

Consulte o proprietário do projeto antes de utilizar o código em um projeto que exija uma licença específica.
