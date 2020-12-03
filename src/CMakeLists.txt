project(accounting)

set(EOSIO_WASM_OLD_BEHAVIOR "Off")
find_package(eosio.cdt)

add_contract( accounting accounting accounting.cpp )
target_include_directories( accounting PUBLIC ${CMAKE_SOURCE_DIR}/../include )
target_ricardian_directory( accounting ${CMAKE_SOURCE_DIR}/../ricardian )