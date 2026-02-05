import asyncio
import sys
from contextlib import asynccontextmanager, AsyncExitStack
from typing import Annotated, List
from fastmcp import FastMCP
from multilspy import LanguageServer
from multilspy.multilspy_config import MultilspyConfig, Language
from multilspy.multilspy_logger import MultilspyLogger

exit_stack = AsyncExitStack()
@asynccontextmanager
async def lsp_lifespan(server: FastMCP):
    """
    Manages the persistent LSP connection.
    This runs ONCE when the MCP server starts.
    """
    global lsp_instance
    
    project_root = "/root/multi-agent" 
    logger = MultilspyLogger()
    config = MultilspyConfig(code_language=Language.GO) 
    lsp_instance = LanguageServer.create(config, logger, project_root)
    
    # Enter the context manager and add it to our stack to keep it alive
    await exit_stack.enter_async_context(lsp_instance.start_server())
    
    try:
        yield  # Server stays here while tools are being called
    finally:
        # 3. Cleanup: This runs when the MCP server is stopped
        print("Shutting down LSP...", file=sys.stderr)
        await exit_stack.aclose()

mcp = FastMCP("PersistentLSP", lifespan=lsp_lifespan)

@mcp.tool()
async def request_definition(
    relative_path: Annotated[str, "Path to the file relative to project root"],
    line: Annotated[int, "0-indexed line number"],
    column: Annotated[int, "0-indexed column number"]
):
    """Find the definition location(s) for the symbol at the given position."""
    return await lsp_instance.request_definition(relative_path, line, column)

@mcp.tool()
async def request_references(
    relative_path: Annotated[str, "Path to the file relative to project root"],
    line: Annotated[int, "0-indexed line number"],
    column: Annotated[int, "0-indexed column number"],
    include_declaration: bool = True
):
    """Find all references (callers/usages) of the symbol at the given position."""
    return await lsp_instance.request_references(relative_path, line, column, include_declaration)

@mcp.tool()
async def request_hover(
    relative_path: Annotated[str, "Path to the file relative to project root"],
    line: Annotated[int, "0-indexed line number"],
    column: Annotated[int, "0-indexed column number"]
):
    """Get documentation and type information (hover card) for the symbol at the given position."""
    return await lsp_instance.request_hover(relative_path, line, column)

@mcp.tool()
async def request_document_symbols(
    relative_path: Annotated[str, "Path to the file relative to project root"]
):
    """Get a list of all symbols (functions, classes, variables) defined in the specified file."""
    return await lsp_instance.request_document_symbols(relative_path)

if __name__ == "__main__":
    mcp.run()