package config

const PageSize = 1004 // Each network request reads the number of files in the directory, the larger the value, the more likely it will time out, and it must not exceed 1000

const RetryLimit = 7     // If a request fails, the maximum number of retries allowed
const ParallelLimit = 20 // The number of parallel network requests can be adjusted according to the network environment

const DefaultTarget = "" // Required, copy the default destination ID, if target is not specified, it will be copied here, it is recommended to fill in the team disk ID

const SaLocation = "sa" // flag to assign

const DBPath = "gdurl.sqlite"