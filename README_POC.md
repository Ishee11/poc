# ***Контекст***

Система — это **журнал финансовых операций игроков в рамках игровой сессии**.
Она **не моделирует игру**, а фиксирует только ввод/вывод денег через фишки.

---

# ***Тип системы***

> ***Ledger / accounting system с кэшируемой проекцией агрегатов***

---

# ***Модель домена***

## Session

Сущность, задающая:

* жизненный цикл: `active → finished`
* курс: `chipRate` (immutable)
* кэш агрегатов:

  * `totalBuyIn`
  * `totalCashOut`

```text
ВАЖНО:
эти поля — НЕ источник истины,
а производная (cached projection) от операций
```

---

## Operation (источник истины)

Факт изменения состояния.

Типы:

* `BuyIn`
* `CashOut`

Атрибуты:

* `operationID` (идемпотентность)
* `sessionID`
* `playerID`
* `chips`
* `createdAt`

👉 **Operations — единственный источник истины**

---

## ChipRate (value object)

```text
chipRate = количество фишек за 1 денежную единицу
```

---

### Свойства

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

### Инвариант

```text
chips % chipRate == 0
```

👉 нельзя вывести дробные деньги

---

## Money (value object)

Представляет денежное значение.

```text
amount = минимальная денежная единица (например копейки)
```

---

### Свойства

```text
amount >= 0
```

---

### Роль в системе

```text
Money:
- НЕ хранится в Operation
- НЕ используется для агрегатов
- является производным значением из chips через ChipRate
- используется при CashOut и итоговых расчётах
```

---

# ***Проекция (aggregates / cache)***

В `Session` хранятся:

```text
totalBuyIn   = SUM(BuyIn)
totalCashOut = SUM(CashOut)
```

```text
tableChips = totalBuyIn − totalCashOut
```

---

## Свойства агрегатов

* обновляются синхронно при записи операций (в одной транзакции)
* всегда согласованы с operations в рамках commit
* могут быть полностью пересчитаны из operations
* используются для быстрых чтений (O(1))
* могут использоваться для предварительных (optimistic) проверок
* **НЕ являются источником истины для финальных бизнес-решений**

---

# ***Ключевые вычисления***

## Стол

```text
tableChips = totalBuyIn − totalCashOut
```

---

## Результат игрока

```text
profitChips = totalCashOut(player) − totalBuyIn(player)
```

```text
profitMoney = profitChips / chipRate
```

---

## CashOut (денежная интерпретация)

```text
cashOutMoney = chips / chipRate
```

---

## Состояние игрока (не хранится)

```text
inGame ⇔ последняя операция игрока = BuyIn
outGame ⇔ CashOut или операций нет
```

👉 определяется только через Operations

---

# ***Инварианты***

## Общие

* операции возможны только при `session.active`
* `chips > 0`
* `operationID` уникален (идемпотентность)

---

## BuyIn

```text
chips > 0
```

---

## CashOut

Проверяется (в UseCase):

### 1. Состояние игрока

```text
lastOperation(player) = BuyIn
```

### 2. Кратность курсу

```text
chips % chipRate == 0
```

(через ChipRate)

---

### 3. Состояние стола

```text
chips <= tableChips
```

```text
ВАЖНО:
финальная проверка выполняется через Operations (repo),
а не через cache Session
```

---

## FinishSession

```text
tableChips == 0
```

---

# ***Транзакционная модель (критично)***

```text
BEGIN

1. проверка инвариантов (Operations + ChipRate)
2. INSERT Operation
3. UPDATE Session.total*

COMMIT
```

---

# ***Гарантии***

* агрегаты всегда согласованы с операциями
* нет состояния, где обновлено только одно
* race conditions предотвращаются

---

# ***Идемпотентность***

```text
operationID уникален
повтор:
- не создаёт дубликат
- не изменяет агрегаты
```

---

# ***Расхождения (Reconciliation)***

```text
реальные фишки ≠ расчётные
```

---

## Контроль

```text
CashOut: chips > tableChips → ошибка
Finish: tableChips > 0 → нельзя завершить
```

---

# ***Восстановление агрегатов***

```text
rebuild Session.total* из Operations
```

---

# ***Границы ответственности***

## Domain

* `Operation` — источник истины
* `Session` — кэш агрегатов + lifecycle
* `ChipRate` — правила конверсии и кратности
* `Money` — денежная интерпретация
* **Session не принимает решения на основе cache**

---

## UseCase

* оркестрация
* транзакции
* вся бизнес-валидация:

  * lastOperation
  * tableChips (через operations)
  * chipRate
* расчёт денежных значений (cashout, итоги)

---

## Repository

* доступ к operations
* транзакции
* блокировки

---

# ***Ключевая мысль***

> *Operations — это единственная правда системы*
> *Session — это кэш для скорости*
> *ChipRate — это правило корректности денег*
> *Money — это интерпретация результата, а не источник состояния*
