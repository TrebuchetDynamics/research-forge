# Quantitative research review: leak-free LightGBM for 5-minute crypto direction prediction

Date: 2026-06-13  
Scope: BTC, ETH, SOL, XRP, DOGE, BNB; 5-minute bars; LightGBM; direction labels; price/volatility/momentum/book/microstructure/cross-timeframe features.  
Trigger: a higher-timeframe aggregation lookahead leak inflated validation AUC from approximately 0.52 to approximately 0.81; after removal, honest out-of-sample AUC is approximately 0.51-0.53.

## Executive assessment

LightGBM is a scientifically defensible model family for this task, but the observed post-leak performance range changes the research question from "how do I tune this?" to "can I falsify the remaining weak signal?" For liquid short-horizon crypto direction, AUC 0.51-0.53 is normal. It may be either noise or weak real signal; it is not, by itself, model failure. It becomes potentially useful only if it is stable across time, venues, assets, label definitions, data vendors, and cost assumptions, and if probability calibration and PnL survive strict chronological/purged validation.

AUC near 0.81 after adding higher-timeframe features is overwhelmingly consistent with leakage or target contamination for this horizon. After leak removal, AUC 0.51-0.53 is more plausible than suspiciously high validation performance.

Bottom line: continue only as a falsification-first research program. Do not promote to production from AUC alone.

## Search basis and references

This report used targeted OpenAlex searches plus known canonical finance/ML references. Key sources:

1. Ke et al. (2017), *LightGBM: A Highly Efficient Gradient Boosting Decision Tree*, NeurIPS. arXiv:1711.07487.
2. Gu, Kelly, and Xiu (2020), *Empirical Asset Pricing via Machine Learning*, Review of Financial Studies. DOI: 10.1093/rfs/hhaa009.
3. Campbell, Lo, and MacKinlay (1997), *The Econometrics of Financial Markets*, Princeton University Press. DOI: 10.1515/9781400830213.
4. Bailey et al. (2014), *Pseudo-Mathematics and Financial Charlatanism: The Effects of Backtest Overfitting on Out-of-Sample Performance*, Notices of the AMS. DOI: 10.1090/noti1105.
5. López de Prado (2018), *Advances in Financial Machine Learning*, Wiley. Canonical source for purging, embargoing, and financial CV methodology.
6. Zhang, Zohren, and Roberts (2019), *DeepLOB: Deep Convolutional Neural Networks for Limit Order Books*, IEEE Transactions on Signal Processing. DOI: 10.1109/TSP.2019.2907260.
7. Ntakaris et al. (2015), *Modelling high-frequency limit order book dynamics with support vector machines*, Quantitative Finance. DOI: 10.1080/14697688.2015.1032546.
8. Sun, Dedahanov, Shin, and Li (2021), *Forecasting and trading cryptocurrencies with machine learning under changing market conditions*, Financial Innovation. DOI: 10.1186/s40854-020-00217-x.
9. Sebastião and Godinho (2021), cryptocurrency price prediction survey, International Journal of Intelligent Systems in Accounting, Finance and Management. DOI: 10.1002/isaf.1488.
10. Lahmiri and Bekiros (2020/2021 related crypto ML literature) and broader crypto forecasting surveys: useful as background, but many crypto prediction papers have weak validation standards relative to production trading requirements.

## A. Literature review

### LightGBM / gradient boosting on financial time series

Gradient-boosted decision trees are appropriate for tabular, heterogeneous, nonlinear features. Financial ML literature supports tree ensembles and boosted trees as competitive baselines when features are cross-sectional, nonlinear, and interaction-heavy. Gu, Kelly, and Xiu show that flexible ML methods can extract weak return-predictive structure in asset pricing, but their conclusions depend on careful out-of-sample evaluation and economically meaningful tests.

For 5-minute crypto direction, LightGBM has advantages:

- handles nonlinear thresholds and interactions;
- works well with mixed price, volatility, momentum, and microstructure features;
- requires less data engineering than sequence models;
- is easier to audit for feature importance and leakage sensitivity than deep models.

But LightGBM does not solve the hardest issues:

- non-stationarity;
- overlapping labels;
- path-dependent execution costs;
- lookahead in feature construction;
- data-snooping from repeated experiments;
- inflated validation from random or improperly purged splits.

### Crypto forecasting literature

Crypto ML papers often report promising directional or trading results, but many use daily data, broad horizons, weak transaction-cost assumptions, or validation schemes that would not satisfy a skeptical market microstructure reviewer. The more credible crypto forecasting results emphasize changing market regimes, instability of learned relationships, and the need for economic validation.

For liquid 5-minute bars, realistic predictive effects are expected to be small. Crypto may be less efficient than mature equities/FX in some periods, but BTC/ETH perpetuals and spot markets are highly competitive. SOL/XRP/DOGE/BNB may show stronger idiosyncratic effects in certain regimes, but those can vanish after costs and regime shifts.

### Order-book and microstructure prediction

DeepLOB and related limit-order-book studies show that order-flow and book-state features can predict short-horizon price movements better than pure OHLCV features, especially at very short horizons. However, the literature also implies strict requirements:

- event-time alignment;
- no future book states in aggregation windows;
- careful treatment of labels around spread/mid-price moves;
- chronological validation;
- realistic latency/slippage/cost modeling.

Microstructure features are among the most plausible survivors after leakage removal, but they are also among the easiest places to introduce accidental lookahead.

### Leakage and overfit literature

Bailey et al. and López de Prado are directly relevant. Finance is unusually vulnerable to backtest overfitting because signal-to-noise is low, experiments are cheap, and researchers repeatedly try labels, horizons, filters, features, splits, and model settings. A single clean holdout is not enough if it has been repeatedly inspected.

## B. Expected performance ranges

Approximate leak-free expectations for liquid 5-minute crypto direction:

| Setup | Plausible AUC range | Interpretation |
| --- | ---: | --- |
| Naive OHLCV/momentum only | 0.500-0.520 | Usually noise or very weak regime effect |
| Good leak-free bar + volatility + cross-asset features | 0.510-0.535 | Plausible weak signal if stable |
| High-quality book/order-flow features, clean alignment | 0.520-0.560 | Plausible, but costs and latency dominate |
| AUC > 0.58 on liquid 5-minute direction | Suspicious | Requires strong independent replication |
| AUC > 0.65 | Very suspicious | Usually leakage, target contamination, split error, duplicate data, or an easy/non-trading label |
| AUC ~0.81 | Near-certain leak | Consistent with your discovered higher-timeframe leak |

AUC 0.51-0.53 can matter in trading only if:

- predictions are calibrated;
- edge concentrates in high-confidence tails;
- turnover is controlled;
- transaction costs, spread, latency, funding, and slippage are included;
- signal is stable enough for live deployment.

## C. Common failure modes

1. **Feature timestamp error**: feature includes data whose bar close occurs after the prediction timestamp.
2. **Higher-timeframe aggregation leak**: 1h/4h/daily candles are joined to 5-minute rows before the higher-timeframe candle has closed.
3. **Rolling calculation leak**: rolling indicators accidentally include the target bar or future bars.
4. **Global normalization**: scalers, z-scores, PCA, encoders, winsorization thresholds, or imputation values fit on all data.
5. **Cross-sectional normalization leak**: using future membership, future volatility, or full-sample asset statistics.
6. **Overlapping target leakage**: train and validation labels share future return intervals without purging/embargoing.
7. **Random split**: destroys time dependence and allows near-duplicate states across train and validation.
8. **Repeated holdout tuning**: validation set becomes training data through researcher decisions.
9. **Survivorship / universe leak**: assets selected because they are currently liquid or successful.
10. **Execution mismatch**: prediction uses close-time features but assumes fills at the same close.
11. **Vendor repair/backfill leak**: historical data corrected or resampled in ways unavailable live.
12. **Book snapshot timing leak**: using book states after the prediction decision time.

## D. Leakage checklist

### Multi-timeframe features

- [ ] Every higher-timeframe feature has an availability timestamp.
- [ ] A 1h candle ending at 10:00 is not available to 5-minute rows before 10:00 plus publication/processing delay.
- [ ] Resampling uses left-closed/right-open windows appropriate to the decision time.
- [ ] Joins use `asof` semantics with `feature_available_at <= decision_time`.
- [ ] Incomplete higher-timeframe candles are either explicitly live-computable or excluded.

### Rolling indicators

- [ ] Rolling windows end strictly before the prediction decision time.
- [ ] Label horizon bars are never included in feature windows.
- [ ] Indicators are shifted after calculation where necessary.
- [ ] Warmup rows are dropped or marked unavailable.

### Normalization

- [ ] Scalers are fit only on training data inside each split.
- [ ] Rolling normalizers use only historical observations.
- [ ] Winsorization/clipping thresholds are train-only or online.
- [ ] Missing-value imputation is train-only or online.
- [ ] Cross-asset ranks/z-scores use only assets and values available at that timestamp.

### Target construction

- [ ] Target return starts after the executable decision time.
- [ ] Entry price assumption is realistic: next bar open, delayed mid, or executable quote, not same-bar close after seeing the close.
- [ ] Label handles spread and fees if the research goal is tradability.
- [ ] Overlapping horizons are purged between train and validation/test.

### Splits

- [ ] No random split for final claims.
- [ ] Validation/test blocks are chronological.
- [ ] Embargo length is at least the maximum label horizon plus feature lookback overlap where needed.
- [ ] The final test set is touched once.
- [ ] All model selection is completed before final test evaluation.

## E. Validation checklist

### Split methods

| Method | Assessment |
| --- | --- |
| Chronological train/validation/test | Minimum acceptable baseline. Good for first honest result. |
| 70/15/15 chronological | Acceptable if dataset is long enough and final test is untouched. |
| 80/20 chronological | Acceptable for a single train/test estimate, but weak for model selection unless validation is internal to train. |
| Walk-forward validation | Stronger. Better reflects retraining and regime change. Preferred for research claims. |
| Rolling retraining | Strong for production simulation if retraining cadence is fixed before evaluation. |
| Purged time-series CV | Academically defensible when labels overlap or feature windows overlap. Add embargo. |
| Random k-fold | Not defensible for final financial time-series claims. |

### Recommended defensible protocol

1. Freeze raw data version, vendor, symbol universe, fees, and timestamp conventions.
2. Define decision time, feature availability time, label interval, and execution price.
3. Use a chronological train/validation/test split for initial audit.
4. Use purged walk-forward validation for model-selection evidence.
5. Keep a final untouched test period, ideally including a recent regime not used in research.
6. Report per-asset and per-regime metrics, not only pooled metrics.
7. Include negative controls:
   - shuffled labels;
   - reversed-time features;
   - deliberately lagged stale features;
   - random features;
   - no-trade baseline;
   - simple momentum/mean-reversion baselines.
8. Require economic metrics after realistic costs.
9. Record every experiment to prevent silent data-snooping.

## F. Feature categories: likely survivors vs likely collapses

### More likely to survive leakage removal

- lagged returns and short-term reversal/momentum, if regime-dependent;
- realized volatility and volatility-of-volatility;
- spread, depth, imbalance, order-flow imbalance, trade-sign imbalance;
- funding/open-interest/liquidation features if timestamped correctly;
- cross-asset lagged information, especially BTC/ETH leading altcoins, if latency is realistic;
- time-of-day/day-of-week effects in crypto, though often weak and unstable;
- regime-state features computed only from past data.

### More likely to collapse after leakage removal

- higher-timeframe OHLCV features joined before candle close;
- centered rolling windows;
- future-filled missing values;
- global z-scores or full-sample volatility normalization;
- indicators computed on finalized candles while pretending to trade intrabar;
- target-derived volatility or threshold labels reused as features;
- cross-timeframe trend flags based on future completed bars;
- features from reconstructed candles unavailable at decision time.

## G. Metrics that matter

| Metric | Usefulness | Limitation |
| --- | --- | --- |
| AUC | Good ranking diagnostic; threshold-independent | Does not prove calibration or profitability |
| Logloss | Rewards calibrated probabilities | Sensitive to class imbalance and label noise |
| Brier score | Good probability-quality metric | Can hide poor tail behavior |
| Calibration curve / ECE | Essential if probabilities drive position sizing | Needs enough samples per bin |
| Sharpe | Economic relevance | Easily overfit; must include costs and realistic sizing |
| PnL/trade | Direct edge measure | Requires execution assumptions; can be dominated by tails |
| Turnover | Cost/risk diagnostic | Not performance alone |
| Win rate | Usually weak | Can be high with bad payoff asymmetry |
| Precision/recall in high-confidence tail | Useful if only trading selective predictions | Tail sample size can be small |

For this project, the minimum metric bundle should be:

- AUC;
- logloss;
- Brier score;
- calibration by decile;
- Sharpe after fees/spread/slippage;
- PnL/trade;
- turnover;
- max drawdown;
- per-asset and per-regime breakdown.

## H. What would convince an academic reviewer there is real signal?

Strong evidence would include:

1. A frozen, timestamp-audited feature pipeline with feature availability times.
2. Purged/embargoed walk-forward validation with predeclared retraining cadence.
3. Final untouched test set performance consistent with validation.
4. Stable AUC/logloss/Brier improvement over naive baselines across multiple assets and regimes.
5. Calibration curves showing monotonic realized outcomes by predicted probability bucket.
6. Economic performance after realistic fees, spread, slippage, and latency.
7. Signal concentration in high-confidence tails, not only tiny average AUC.
8. Robustness to reasonable label and horizon variations.
9. Negative controls that collapse to random.
10. Reproducible experiment logs and no manual post-hoc exclusion of bad periods.

## I. What would convince a reviewer it is still overfit or leaking?

Red flags:

1. Large gap between validation and final test.
2. AUC jumps after adding any feature family without an availability-time explanation.
3. Feature importance dominated by higher-timeframe or normalized features.
4. Performance disappears under one-bar feature lag.
5. Performance disappears under purging/embargoing.
6. Calibration is non-monotonic or unstable.
7. Results depend on a single exchange, asset, or short period.
8. Sharpe vanishes after fees or small latency.
9. Too many tried variants without multiple-testing correction or experiment ledger.
10. Final test was inspected repeatedly during development.

## J. Assessment of AUC 0.51-0.53

### Noise?

Possible. AUC 0.52 can arise from residual dependence, class imbalance artifacts, or researcher degrees of freedom. Treat noise as the default null hypothesis.

### Weak but real signal?

Also plausible. In liquid short-horizon markets, real signals are often tiny. A stable AUC 0.52 with good calibration, tail concentration, and positive cost-adjusted PnL can be valuable.

### Model failure?

Not necessarily. LightGBM returning AUC 0.51-0.53 after leakage removal may indicate the model is now seeing the true difficulty of the task. Model failure would be more likely if simpler baselines beat it consistently, calibration is poor, or feature importance is unstable across folds.

Final classification: **weak-but-possibly-real signal, not yet proven**. Academic default: **unproven until it survives purged walk-forward validation, calibration checks, negative controls, and cost-adjusted trading simulation**.

## K. Recommended research roadmap

Do not start with new features or hyperparameter tuning. Start with falsification.

### Phase 1: Timestamp and leakage audit

- Define `event_time`, `bar_open`, `bar_close`, `feature_available_at`, `decision_time`, `label_start`, `label_end`, and `execution_time`.
- Add automated assertions that no feature has `feature_available_at > decision_time`.
- Unit-test every multi-timeframe join.
- Add one-bar and one-window lag stress tests.

### Phase 2: Rebuild validation

- Freeze a chronological 70/15/15 or 80/10/10 split.
- Add purged walk-forward validation.
- Embargo at least the maximum label horizon and any overlapping feature/label contamination window.
- Keep one final test set untouched.

### Phase 3: Baselines and negative controls

- Compare against constant probability, lagged return, volatility-only, and simple momentum/reversal baselines.
- Run shuffled-label and time-shifted-feature controls.
- Require all controls to collapse to AUC near 0.50 and zero economic value.

### Phase 4: Probability and economics

- Report logloss, Brier, calibration, and AUC together.
- Trade only predeclared probability thresholds.
- Include fees, spread, slippage, latency, turnover, and capacity.
- Report per-asset and per-regime results.

### Phase 5: Production-gate standard

Promote only if:

- final untouched test AUC/logloss/Brier improve over baselines;
- calibration is monotonic and stable;
- cost-adjusted PnL is positive after conservative costs;
- performance survives walk-forward retraining;
- no single feature family or period explains all performance;
- experiment history is auditable.

## Conclusion

LightGBM is appropriate as an auditable tabular baseline for 5-minute crypto direction prediction. The old AUC ~0.81 was not credible after the discovered higher-timeframe leak. The current AUC ~0.51-0.53 is scientifically plausible and may be useful, but only under strict evidence standards. The next correct step is not tuning; it is proving that the remaining edge survives leakage audits, purged walk-forward validation, negative controls, calibration tests, and realistic execution costs.
