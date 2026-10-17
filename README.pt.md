[Русский](README.md) | [English](README.en.md) | [Español](README.es.md) | [**Português**](README.pt.md)

![Gerenciador de mods para GTA San Andreas](assets/banner.png)

![Так выглядит окно](assets/screenshot.png)

# Gerenciador de mods para GTA San Andreas

Programa para Windows que mantém uma instalação com mods em ordem: um interruptor para cada mod, aviso quando dois mods usam o mesmo arquivo e perfis de configuração salvos.

O problema clássico: você instala cinco mods, o jogo não abre e o culpado é um único `infernus.dff` colocado por dois mods ao mesmo tempo. O gerenciador mostra esses choques antes de abrir o jogo.

**Baixar:** [sa-mod-manager-windows.zip](https://github.com/hilsetapp/sa-mod-manager/releases/latest/download/sa-mod-manager-windows.zip)

## O que faz

- **Lista de mods com interruptores.** Um mod fica em `modloader` (ligado) ou em `modmanager/off` (desligado). Ligar apenas move a pasta: nada é apagado e tudo é reversível.
- **Busca de conflitos.** Se dois mods trazem um arquivo com o mesmo nome, o gerenciador mostra quem está disputando e qual arquivo, separando o que quebra o jogo agora do que é só um aviso.
- **Marca de arquivo original.** Arquivos que já existem em `models/gta3.img` aparecem marcados como "substitui um arquivo do jogo".
- **Avisos sobre arquivos suspeitos.** Substituição completa do `gta3.img`, um `d3d9.dll` desconhecido, um `.exe` dentro do mod: melhor saber antes.
- **Perfis de configuração.** Salve o conjunto de mods ligados ("SA-MP", "modo história") e troque entre eles com um clique.
- **Diário.** Tudo o que o gerenciador faz é escrito na janela e em `modmanager/journal.log`.
- **Importar de zip.** Escolha o arquivo, ele é extraído na pasta de mods desligados e você liga quando quiser.
- **Funciona com instalações manuais.** Mods que você já colocou na mão em `modloader` aparecem sozinhos na lista.

## Como começar

1. Baixe `sa-mod-manager-windows.zip` e extraia em qualquer pasta, de preferência fora de `Program Files`.
2. Execute `sa-mods.exe`. Se o Windows mostrar o aviso azul do SmartScreen, clique em "Mais informações" e depois em "Executar assim mesmo". O programa não baixa nada.
3. Clique em **"Выбрать папку"** (escolher pasta) e aponte para a pasta onde está o `gta_sa.exe`, normalmente `C:\Games\GTA San Andreas`.
4. Clique em **"Добавить мод из архива"** (adicionar mod do arquivo) e escolha o zip do mod: ele aparece desligado.
5. Ligue o interruptor e abra o jogo.

A interface está em russo: a ferramenta é feita primeiro para a cena de San Andreas em russo. Jogadores brasileiros são muito bem-vindos, a tradução da interface está na lista de tarefas.

Atenção: para o jogo enxergar a pasta `modloader` é preciso ter o **ModLoader** (`modloader.asi`) instalado. Se não tiver, o gerenciador avisa no topo da janela, porque ligar mods sem ModLoader não muda nada no jogo.

## Linha de comando

`sa-mods-cli.exe` faz o mesmo sem janela, bom para scripts:

```
sa-mods-cli.exe -game "C:\Games\GTA San Andreas" -check
sa-mods-cli.exe -game "C:\Games\GTA San Andreas" -list
sa-mods-cli.exe -game "C:\Games\GTA San Andreas" -conflicts
sa-mods-cli.exe -game "C:\Games\GTA San Andreas" -import C:\Downloads\silentpatch.zip
```

Opções: `-check` o que existe na pasta do jogo, `-list` mods, `-conflicts` choques, `-import` adicionar zip, `-enable` e `-disable` ligar ou desligar pelo nome da pasta.

## Como funciona por dentro

- O San Andreas reconhece arquivos pelo nome. Se dois mods trazem `infernus.dff`, a última pasta vence e um dos mods para de funcionar sem avisar. Por isso um conflito aqui é "mesmo nome de arquivo em dois mods", e em arquivos de dados (`.cfg`, `.dat`, `.ipl`) "mesmo caminho relativo".
- Scripts CLEO (`.cs`), plugins ASI (`.asi`) e `.fxt` também são comparados pelo nome.
- Ligar e desligar move a pasta entre `modloader` e `modmanager/off`. O gerenciador não altera o jogo, não escreve nos arquivos `.img` e não mexe nos seus arquivos.
- Os nomes dentro de `models/gta3.img` são lidos só para saber qual arquivo do jogo o mod substitui.

## O que não faz

- Não executa instaladores `.exe` e não edita scripts de terceiros.
- Não compara o conteúdo dos modelos, só os nomes, igual ao jogo.
- Não baixa mods e não tem catálogo de links.
- Não é app de navegador nem de celular: é um programa de desktop para Windows.

## Compilar

Precisa de Go 1.26 ou mais novo, sem compilador de C:

```
go test ./...
go build -ldflags "-H=windowsgui -s -w" -o sa-mods.exe ./cmd/sa-mods
go build -s -w -o sa-mods-cli.exe ./cmd/sa-mods-cli
```

## Licença

MIT. A fonte Inter usa a licença SIL Open Font License 1.1, detalhes em [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).
