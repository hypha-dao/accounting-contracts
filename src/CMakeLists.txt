project(accounting)

set(EOSIO_WASM_OLD_BEHAVIOR "Off")
find_package(eosio.cdt)

set (DOC_GRAPH_DIR ${CMAKE_SOURCE_DIR}/../document-graph)

add_contract( accounting accounting 
accounting.cpp
${DOC_GRAPH_DIR}/src/document_graph/util.cpp
${DOC_GRAPH_DIR}/src/document_graph/content.cpp
${DOC_GRAPH_DIR}/src/document_graph/content_wrapper.cpp
${DOC_GRAPH_DIR}/src/document_graph/document.cpp
${DOC_GRAPH_DIR}/src/document_graph/document_graph.cpp
${DOC_GRAPH_DIR}/src/document_graph/edge.cpp
               )

target_include_directories( accounting PUBLIC 
                            ${CMAKE_SOURCE_DIR}/../include 
                            ${DOC_GRAPH_DIR}/include)
target_ricardian_directory( accounting ${CMAKE_SOURCE_DIR}/../ricardian )