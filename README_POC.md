# ***Контекст***

Система — это **журнал (ledger) операций с фишками в рамках игровой сессии**.

Система:

* не моделирует игровой процесс
* не хранит баланс игрока как состояние
* фиксирует только **факты изменений (operations)**

---

# ***Тип системы***

```text
Append-only ledger + transactional cached projection
```

---

# ***Модель домена***

## Operation (source of truth)

Факт изменения состояния системы.

```text
Operation — единственный источник истины
```

Типы:

* `BuyIn`
* `CashOut`
* `Reversal`

Атрибуты:

* `operationID` — внутренний идентификатор
* `requestID` — идемпотентность (внешний)
* `sessionID`
* `playerID`
* `chips`
* `createdAt`
* `referenceID` (для reversal)

---

### Свойства

```text
- операции не изменяются
- операции не удаляются
- любые изменения — только через новые операции
```

---

## Reversal (compensation)

Компенсирующая операция.

```text
Reversal:
- не удаляет исходную операцию
- создаёт новую запись
- инвертирует эффект
```

Инверсия:

```text
BuyIn   → CashOut
CashOut → BuyIn
```

---

### Инварианты

```text
- нельзя делать reversal для reversal
- reversal только один раз для операции
- reversal только при active session
```

---

## Session

Сущность жизненного цикла и кэша агрегатов.

```text
Session:
- управляет lifecycle
- хранит cached aggregates
```

Поля:

* `status: active | finished`
* `chipRate` (immutable)
* `totalBuyInCache`
* `totalCashOutCache`

---

### Важно

```text
Session aggregates:
- являются cached projection от Operations
- обновляются синхронно в write-path
- являются частью консистентного состояния в транзакции
- НЕ являются источником истины
```

---

## ChipRate (value object)

```text
chipRate = количество фишек за 1 денежную единицу
```

---

### Инварианты

```text
chipRate > 0
```

---

### Поведение

```text
money → chips: chips = money * chipRate
chips → money: money = chips / chipRate
```

---

### Ограничение

```text
chips % chipRate == 0
```

---

## Money (value object)

```text
amount = минимальная денежная единица
```

---

### Роль

```text
Money:
- не хранится в Operation
- вычисляется из chips через ChipRate
- используется только для интерпретации результата
```

---

# ***Проекция (aggregates / cache)***

```text
totalBuyIn   = SUM(BuyIn)
totalCashOut = SUM(CashOut)
```

```text
tableChips = totalBuyIn − totalCashOut
```

---

### Свойства

```text
- обновляются в той же транзакции, что и Operation
- всегда согласованы при commit
- могут быть пересчитаны из Operations
- используются для быстрых чтений
```

---

### Ограничение

```text
финальные проверки выполняются через repo (Operations),
а не через Session cache
```

---

# ***Состояние игрока***

Не хранится явно.

```text
определяется через Operations
```

---

### Правило

```text
inGame  ⇔ последняя эффективная операция = BuyIn
outGame ⇔ иначе
```

```text
учитываются reversal (эффективное состояние)
```

---

# ***Идемпотентность***

```text
идемпотентность обеспечивается через requestID
```

---

### Контракт

```text
requestID:
- приходит извне
- уникален на уровне клиента
- при повторе операция не создаётся
```

```text
operationID:
- генерируется системой
- не участвует в идемпотентности
```

---

# ***Ключевые вычисления***

## Стол

```text
tableChips = totalBuyIn − totalCashOut
```

---

## Игрок

```text
profitChips = cashOut − buyIn
profitMoney = profitChips / chipRate
```

---

## CashOut

```text
cashOutMoney = chips / chipRate
```

---

# ***Инварианты***

## Общие

```text
- session.status = active
- chips > 0
- requestID валиден
```

---

## BuyIn

```text
chips > 0
```

---

## CashOut

Проверяется через repo (operations):

```text
1. lastOperation(player) = BuyIn
2. chips % chipRate == 0
3. chips <= tableChips
```

---

## Reversal

```text
- target существует
- target не reversal
- reversal ещё не выполнялся
- session active
```

---

## FinishSession

```text
tableChips == 0
```

(через repo, не через cache)

---

# ***Транзакционная модель***

Контракт usecase:

```text
TxManager.RunInTx(fn)
```

---

## Порядок выполнения

```text
BEGIN

1. идемпотентность (requestID)
2. загрузка данных (session, operations)
3. проверки (инварианты)
4. создание Operation
5. обновление Session cache
6. сохранение Operation + Session

COMMIT
```

---

### Гарантии

```text
- атомарность
- согласованность aggregates и operations
```

---

### Ограничение

```text
race conditions предотвращаются только
при корректной реализации repo (locking / isolation level)
```

---

# ***Роль слоёв***

## Domain

```text
Operation:
- источник истины

Session:
- lifecycle
- cached aggregates

ChipRate:
- правила конверсии

Money:
- интерпретация
```

---

## UseCase

```text
- оркестрация
- транзакции
- вся бизнес-валидация
- идемпотентность
```

---

## Repository

```text
- доступ к operations
- агрегаты (read-model)
- транзакции
- блокировки
```

---

# ***Восстановление***

```text
Session aggregates могут быть полностью пересчитаны из Operations
```

---

# ***Контроль расхождений***

```text
CashOut:
chips > tableChips → ошибка

Finish:
tableChips > 0 → нельзя завершить
```

---

# ***Ключевая мысль***

```text
Operations — единственный источник истины
Session — консистентный cache внутри транзакции
Reversal — единственный способ изменить прошлое
Money — производная интерпретация
```

---
сделать:
в списке операций не нужно пользователю видеть айди операции. давай кнопку отмены операции вернем как было без красной подсветки, только в саммом подтверждении оставим красное. результаты игроков в виде таблицы не удается сделать? в мобильной версии имею ввиду. давай попробуем шрифт поменьше там сделать, чтоб поместилось, проверим как будет выглядеть.

чип рейт - при создании по умолчанию (если не вели) ставить 2 и выводить как при завершении сессии подтверждение с текстом о том с какими параметрами чип рейта будет создана сессия, еще нужна возможность его изменения в еще активной сессии

по умолчанию сделать добавление 1000 фишек если не ввели ничего
выбор игрока для байина (раскрывающийся список), если можно прям в раскрывающемся списке кнопку добавления нового игрока сверху
