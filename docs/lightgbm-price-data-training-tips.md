# LightGBM price-data training tips

ResearchForge project used: `tmp/lightgbm-training-research/project`  
Workflow evidence: OpenAlex searches, JSON import, extraction schema `training-guidance`, accepted evidence items, and clean `rforge evidence audit`.

## Scope

Use case: LightGBM for short-horizon price-direction prediction on liquid crypto or other high-frequency price data.

This is training guidance, not a promise of profitable signal. The central rule is: **prove the pipeline is leak-free before optimizing the model**.

## Research basis

ResearchForge queried OpenAlex and imported 18 source records. Evidence items were recorded against these key sources:

- Ke et al. (2017), *LightGBM: A Highly Efficient Gradient Boosting Decision Tree*.
- Gu, Kelly, and Xiu (2020), *Empirical Asset Pricing via Machine Learning*, DOI `10.1093/rfs/hhaa009`.
- Sun et al. (2021), *Forecasting and trading cryptocurrencies with machine learning under changing market conditions*, DOI `10.1186/s40854-020-00217-x`.
- Zhang, Zohren, and Roberts (2019), *DeepLOB: Deep Convolutional Neural Networks for Limit Order Books*, DOI `10.1109/tsp.2019.2907260`.
- Bailey et al. (2014), *Pseudo-Mathematics and Financial Charlatanism: The Effects of Backtest Overfitting on Out-of-Sample Performance*, DOI `10.1090/noti1105`.
- Ntakaris et al. (2015), *Modelling high-frequency limit order book dynamics with support vector machines*, DOI `10.1080/14697688.2015.1032546`.

## 1. Timestamp discipline before training

Define these fields before feature generation:

```text
bar_open_time
bar_close_time
feature_available_at
decision_time
label_start_time
label_end_time
execution_time
```

Hard invariant:

```text
feature_available_at <= decision_time < label_start_time <= label_end_time
```

For 5-minute bars, a conservative setup is:

```text
features: bars fully closed through t
decision: after close t
entry: next bar open, next mid, or next executable quote
label: return from entry to future horizon
```

Do not compute features using a close and then assume execution at that same close unless the live system can actually see and trade that price.

## 2. Validation design

Do not use random splits for final claims.

Minimum defensible setup:

```text
train: chronological first 70%
validation: next 15%
test: final 15%, untouched
```

Preferred setup:

```text
purged walk-forward validation
fixed retraining cadence
final untouched test period
```

If labels overlap, embargo at least the label horizon. If feature lookbacks are long or multi-timeframe features are used, increase the embargo accordingly.

## 3. LightGBM baseline parameters

Start conservative:

```python
params = {
    "objective": "binary",
    "metric": ["auc", "binary_logloss"],
    "boosting_type": "gbdt",
    "learning_rate": 0.02,
    "num_leaves": 15,
    "max_depth": 4,
    "min_data_in_leaf": 500,
    "feature_fraction": 0.7,
    "bagging_fraction": 0.7,
    "bagging_freq": 1,
    "lambda_l1": 1.0,
    "lambda_l2": 5.0,
    "max_bin": 255,
    "verbosity": -1,
}
```

For noisy price data, prefer:

- shallow trees;
- large `min_data_in_leaf`;
- regularization;
- early stopping;
- fewer hyperparameter searches.

Avoid starting with:

- unlimited depth;
- very high `num_leaves`;
- tiny leaves;
- heavy validation-set tuning;
- feature selection based on the final test period.

## 4. Expected performance

For leak-free short-horizon liquid-market direction prediction:

| AUC | Interpretation |
| --- | --- |
| `0.500-0.510` | likely noise |
| `0.510-0.530` | normal weak-signal range; possibly useful if stable |
| `0.530-0.560` | interesting, must be stress-tested |
| `>0.580` | suspicious unless independently replicated |
| `>0.650` | likely leakage or target contamination |
| `~0.800` | almost certainly leakage for liquid 5-minute direction |

AUC `0.52` is not automatically bad. It is plausible. It is also not enough.

## 5. Features most likely to survive leakage removal

More plausible:

- lagged returns;
- short-term realized volatility;
- volume/trade imbalance;
- order-book imbalance;
- spread and depth;
- lagged BTC/ETH information for altcoins;
- funding/open interest/liquidations if timestamped correctly;
- time-of-day and day-of-week effects;
- regime features computed only from past data.

Often leakage-prone:

- higher-timeframe OHLCV joined before the higher timeframe closes;
- centered rolling windows;
- global z-scores or full-sample normalization;
- future-filled missing values;
- target-derived volatility or thresholds reused as features;
- same-bar labels that assume impossible execution;
- book snapshots captured after decision time.

## 6. Negative controls

Run these before tuning:

| Control | Expected result |
| --- | --- |
| shuffled labels | AUC near `0.50` |
| random features | AUC near `0.50` |
| extra one-bar feature lag | weaker but not pathological result |
| random split comparison | ignored for claims, useful only to expose leakage sensitivity |
| feature-family ablation | no single suspicious family should explain all signal |

If shuffled labels produce persistent AUC above about `0.51`, assume a pipeline or validation problem.

## 7. Metrics to report

Do not rely on AUC alone. Report:

- AUC;
- binary logloss;
- Brier score;
- calibration by probability bucket;
- precision/return in high-confidence tails;
- Sharpe after fees/spread/slippage;
- PnL per trade;
- turnover;
- max drawdown;
- per-asset and per-regime results.

A model with AUC `0.52` can matter only if probability buckets are calibrated and cost-adjusted PnL concentrates where the model is confident.

## 8. Training workflow

Recommended sequence:

```text
1. Freeze raw data and timestamp semantics.
2. Build a simple leak-free feature set.
3. Train conservative LightGBM.
4. Run shuffled-label and random-feature controls.
5. Run chronological validation.
6. Run purged walk-forward validation.
7. Check calibration and probability buckets.
8. Simulate execution after fees, spread, slippage, and latency.
9. Only then tune hyperparameters.
```

## 9. Practical tuning order

Tune only after controls pass.

Suggested order:

1. `min_data_in_leaf` and `max_depth` to control overfit.
2. `num_leaves` consistent with `max_depth`.
3. `lambda_l1`, `lambda_l2`.
4. `feature_fraction`, `bagging_fraction`.
5. `learning_rate` and number of rounds.
6. Probability calibration on validation predictions if needed.

Stop tuning if validation gains do not survive walk-forward folds.

## 10. Promotion checklist

Do not promote unless:

- no feature violates availability-time checks;
- chronological validation and purged walk-forward agree directionally;
- final test was not used for iteration;
- negative controls collapse to random;
- calibration is monotonic enough for sizing;
- cost-adjusted PnL is positive after conservative costs;
- performance is not isolated to one asset or one regime;
- experiment history is reproducible.

## Bottom line

LightGBM is a good tabular baseline for price, volatility, and microstructure features. For 5-minute crypto direction, honest signal is usually small. The goal is not to tune AUC from `0.52` to `0.54`; the goal is to prove `0.52` is real, stable, calibrated, and tradable after costs.
