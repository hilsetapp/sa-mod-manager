# Самопроверка менеджера: собирает тестовую папку игры и прогоняет по ней
# командную строку. Если что-то не так, скрипт падает с ошибкой, а не молчит.
#
# Запуск:  powershell -ExecutionPolicy Bypass -File scripts\selftest.ps1
param(
    [string]$Cli = ".\sa-mods-cli.exe",
    [switch]$Keep
)

# Вывод программы сравниваем в UTF-8: иначе Windows подставит свою
# кодировку консоли, и русские строки не совпадут.
[Console]::OutputEncoding = [Text.Encoding]::UTF8
$OutputEncoding = [Text.Encoding]::UTF8

$ErrorActionPreference = "Stop"

if (-not (Test-Path $Cli)) { throw "Не нахожу $Cli - собери проект или укажи путь через -Cli" }
$Cli = (Resolve-Path $Cli).Path

function Fail($text) { throw "САМОПРОВЕРКА НЕ ПРОШЛА: $text" }
function Ok($text)   { Write-Host "  [ок] $text" }

$dir = Join-Path $env:TEMP ("sa-mods-test-" + [guid]::NewGuid().ToString("N").Substring(0, 8))
New-Item -ItemType Directory -Force -Path (Join-Path $dir "models") | Out-Null
New-Item -ItemType Directory -Force -Path (Join-Path $dir "modloader") | Out-Null
Write-Host "Тестовая папка: $dir"

try {
    # 1. файлы игры
    "" | Out-File -Encoding ascii (Join-Path $dir "gta_sa.exe")
    "" | Out-File -Encoding ascii (Join-Path $dir "modloader.asi")

    # 2. маленький, но настоящий архив gta3.img: подпись VER2, каталог в блоке 1
    $img = New-Object System.Collections.Generic.List[byte]
    $img.AddRange([Text.Encoding]::ASCII.GetBytes("VER2"))
    $img.AddRange([BitConverter]::GetBytes([uint32]1))   # один файл
    $img.AddRange([BitConverter]::GetBytes([uint32]1))   # каталог в блоке 1 (2048 байт)
    while ($img.Count -lt 2048) { $img.Add(0) }
    $entry = New-Object byte[] 32
    $name = [Text.Encoding]::ASCII.GetBytes("infernus.dff")
    [Array]::Copy($name, 0, $entry, 8, $name.Length)
    $img.AddRange($entry)
    [IO.File]::WriteAllBytes((Join-Path $dir "models\gta3.img"), $img.ToArray())

    # 3. два мода, которые делят один файл
    foreach ($mod in @("Тюнинг тачек", "Пак машин")) {
        $p = Join-Path $dir "modloader\$mod"
        New-Item -ItemType Directory -Force -Path $p | Out-Null
        "" | Out-File -Encoding ascii (Join-Path $p "infernus.dff")
    }

    Write-Host "Проверяю папку игры"
    $check = & $Cli -game $dir -check
    if ($LASTEXITCODE -ne 0) { Fail "ключ -check вернул код $LASTEXITCODE" }
    if (-not ($check -match "похоже на GTA SA: да")) { Fail "менеджер не признал тестовую папку игрой`n$check" }
    if (-not ($check -match "имена в архиве: 1")) { Fail "архив gta3.img не разобрался`n$check" }
    Ok "папка игры читается, имена из gta3.img нашлись"

    Write-Host "Ищу конфликты"
    $conf = & $Cli -game $dir -conflicts
    if (-not ($conf -match "infernus.dff")) { Fail "конфликт по infernus.dff не найден`n$conf" }
    if (-not ($conf -match "СЕЙЧАС МЕШАЮТ")) { Fail "менеджер не понял, что моды мешают прямо сейчас`n$conf" }
    Ok "конфликт видно"

    Write-Host "Добавляю мод из архива"
    $zipSrc = Join-Path $dir "Ночной город"
    New-Item -ItemType Directory -Force -Path (Join-Path $zipSrc "models") | Out-Null
    "" | Out-File -Encoding ascii (Join-Path $zipSrc "models\sultan.dff")
    $zip = Join-Path $env:TEMP ("sa-mods-mod-" + [guid]::NewGuid().ToString("N").Substring(0, 8) + ".zip")
    Compress-Archive -Path $zipSrc -DestinationPath $zip

    $imp = & $Cli -game $dir -import $zip
    if ($LASTEXITCODE -ne 0) { Fail "импорт архива вернул код $LASTEXITCODE`n$imp" }
    $list = & $Cli -game $dir -list
    if (-not ($list -match "Ночной город")) { Fail "добавленный мод не появился в списке`n$list" }

    # 4. включение и выключение двигает папку, а не копирует
    & $Cli -game $dir -enable "ночной_город" | Out-Null
    $onPath = Join-Path $dir "modloader\ночной_город"
    if (-not (Test-Path $onPath)) { Fail "после включения мод не встал в modloader" }
    & $Cli -game $dir -disable "ночной_город" | Out-Null
    $offPath = Join-Path $dir "modmanager\off\ночной_город"
    if (-not (Test-Path $offPath)) { Fail "после выключения мод не вернулся в modmanager\off" }
    Ok "включение и выключение мода работают"

    Write-Host "САМОПРОВЕРКА ПРОШЛА"
    exit 0
}
catch {
    Write-Host $_.Exception.Message
    exit 1
}
finally {
    if (-not $Keep) { Remove-Item -Recurse -Force $dir -ErrorAction SilentlyContinue }
}
