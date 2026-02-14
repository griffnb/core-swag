package swag

// REMOVED: TestListPackages and TestGetAllGoFileInfoFromDepsByList
// These tests were for internal implementation details (listOnePackages, getAllGoFileInfoFromDepsByList)
// that have been moved to internal/loader package during the LoaderService refactoring.
// The functionality is still tested via integration tests like TestCoreModelsIntegration.
// If loader-specific tests are needed, they should be added to internal/loader/loader_test.go
