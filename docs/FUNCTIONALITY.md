# Functionality verification matrix

This maps shipped behavior to automated evidence. Results from the reproducible
run are in [results/README.md](results/README.md); individual passing test names
are recorded in [results/results.json](results/results.json). “Automated” does
not mean every edge case or interactive browser behavior has been verified.

| Functionality | Evidence | Mode / limitation |
| --- | --- | --- |
| Welcome, own-data setup, example download | TestFirstRunChoiceAndPersistence; TestSetupOnlyOffersLogInputAndOptionalJira; packaged smoke | Handler + executable HTTP |
| Offline demo, one alert and six contributing rows | TestDemoPersistenceExportAndIsolation; packaged smoke | Automated synthetic |
| Six failures, inclusive 10-minute boundary | TestDefaultRuleSyntheticScenarios; packaged CLI smoke | Exact known inputs |
| Below threshold, expired window, generic ERROR | Same synthetic scenarios; TestWindowBoundariesAndRearm | Automated negative cases |
| IP normalization, attribution, separate sources | TestIPNormalizationAndExactMessage; TestInvalidAndUnattributed | IPv4/IPv6 synthetic |
| Malformed JSON, missing/wrong-type fields | TestJSONLFieldBoundaries; TestValidation; FuzzParseLine | Defined regressions + bounded fuzz |
| Out-of-order input and rejection accounting | TestOrderedRejectionsAndLineAccounting; TestInvalidAndUnattributed | Rejects rather than sorts |
| IP/alert/line limits, cancellation, read failures | TestLimitsAndCancellation; TestReadFailureAndCancellationDuringInput; TestIncompleteResultsVisible | Partial status and reason; cannot interrupt arbitrary blocking Read |
| Sensitivity counts, boundaries and evidence scope | TestSensitivityScopeAndInclusiveBoundaries; TestSensitivityBounds; TestSensitivityMatchesIndependentIntervalCounts | Evidence-only; not a full-input rerun |
| Slow/distributed/benign replay behavior | TestReplayDocumentsBlindSpots; TestReplayCLIAndUI; packaged smoke | Four synthetic scenarios, not field accuracy |
| Upload, JSON export and persisted latest report | TestOwnLogUploadAndCompletionBoundary; TestDemoPersistenceExportAndIsolation; packaged restart smoke | Actual multipart upload and executable restart |
| Invalid actions/save failure preserve prior report | TestInvalidActionsPreserveLatestReport | Forced local failure |
| Concurrent requests | TestConcurrentExportsAndDemo; race suite | Defined concurrency cases |
| Legacy sensitivity-free reports | TestLegacyReportSensitivityHint | Handler compatibility; not every historical schema |
| Guide restart/completion persistence | TestFirstRunChoiceAndPersistence; TestPreferenceFailures; packaged smoke | Does not replace saved results |
| White/black theme preference | TestThemePreference; TestThemesAcrossPagesAndProtectedPreference; packaged smoke | Cookies/HTML verified; visual appearance still manual |
| Token, host, origin, method, sample and redirect boundaries | TestRequestBoundaries; TestRequestAndSampleBoundaries; theme tests | Local request security tests |
| Escaped stored text | TestSavedTicketTextEscaped | Defined XSS regression |
| Demo cannot send Jira tickets | TestDemoPersistenceExportAndIsolation; demo form assertions | Server-side rejection before handoff |
| Jira read-only check and credential non-retention | TestJiraReadOnlyAndNoCredentialRetention | Local TLS fake server only |
| Jira rejection/retry, lost response, duplicate-create protection | TestHandoffFailureMatrix; TestLostReplyAndRedirect; TestConcurrentHandoffsCreateOnlyOnce | Mocked; no live Jira |
| Persisted intent, explicit link recovery, origin aliases | TestPendingIntentReconcilesByExplicitLink; TestOriginAliasesShareDuplicateProtection; TestOriginPortsAndConflictingHistory | Mocked/network-disabled |
| Single-instance locking | Packaged smoke duplicate launcher | Disposable shared data directory |
| Processing speed and allocation volume | BenchmarkProcess; results/benchmarks.txt | Synthetic in-memory inputs; not peak RSS or UI latency |
| Windows execution | Native race/tests/build/packaged smoke | This run's OS is recorded in results |
| Linux build | Cross-build | Not local Linux execution; see CI when it completes |
| Browser layout, keyboard/zoom, screen reader, OS download warning | TESTING.md checklist | **Manual: not observed** |
| Live Jira permissions/custom mandatory fields | Not run | **Unverified; no credentials or live requests** |

A complete report means the defined input was processed, not that the environment
is safe. Synthetic benchmarks and statement coverage cannot establish operational
security effectiveness. No result in this matrix represents production adoption.
